package workflow

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

var (
	indexEntries = []struct{ Name, Code string }{
		{"道琼斯", "^DJI.US"},
		{"纳斯达克", "^IXIC.US"},
		{"上证指数", "000001.SH"},
		{"深证成指", "399001.SZ"},
		{"恒生指数", "800000.HK"},
	}
	newsMarkets    = []string{"US", "CN", "HK"}
	tradingDayCode = "00700.HK"
)

// PhaseASteps returns pre-market phase A workflow.
func PhaseASteps() []Step {
	steps := []Step{
		{Name: "check_trading_day", Tool: "check_trading_day", Arguments: map[string]any{"code": tradingDayCode}},
		{Name: "get_report_bot_codes", Tool: "get_report_bot_codes"},
	}
	for _, idx := range indexEntries {
		steps = append(steps, Step{
			Name: "index_" + idx.Code,
			Tool: "get_mcp_analysis",
			Arguments: map[string]any{
				"name": idx.Name, "code": idx.Code, "prompt_id": indexPromptID, "period": "hourly", "language": "cn",
			},
		})
	}
	for _, market := range newsMarkets {
		steps = append(steps, Step{
			Name:      "market_news_" + strings.ToLower(market),
			Tool:      "fetch_market_news",
			Arguments: map[string]any{"market": market, "limit": 8},
		})
	}
	steps = append(steps, Step{
		Name: "phase_a_complete", Tool: "write_execution_log",
		ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"step":    "phase_a_complete",
				"message": fmt.Sprintf("indices_done=%v market_news_done=%v", w.MarketContext.IndicesDone, w.MarketContext.MarketNewsDone),
				"status":  "ok",
			}
		},
	})
	return steps
}

// PerStockSteps returns phase B steps for each bot stock.
func PerStockSteps() []Step {
	return []Step{
		{Name: "list_today_reports", Tool: "list_today_reports", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{"code": w.CurrentStock}
		}},
		{Name: "stock_news", Tool: "fetch_stock_news", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{"code": w.CurrentStock, "stock_name": ws.StockName, "limit": 5}
		}},
		{Name: "capital_flow", Tool: "get_capital_flow", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{"code": w.CurrentStock, "period": "DAY"}
		}},
		{Name: "capital_distribution", Tool: "get_capital_distribution", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{"code": w.CurrentStock}
		}},
		{Name: "weekly_analysis", Tool: "get_mcp_analysis", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{
				"name": ws.StockName, "code": w.CurrentStock, "prompt_id": indexPromptID, "period": "weekly", "language": "cn",
			}
		}},
		{Name: "bot_attitude", Tool: "get_bot_yesterday_attitude", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{"bot_id": ws.BotID, "code": w.CurrentStock, "language": "cn"}
		}},
		{Name: "save_local_report", Tool: "save_local_report", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"code": w.CurrentStock, "content": BuildReportContent(w, w.CurrentStock), "report_type": "premarket",
			}
		}},
		{Name: "create_pre_market_report", Tool: "create_pre_market_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreateReportArgsContext(ctx, w, w.CurrentStock)
		}},
		{Name: "stock_complete", Tool: "write_execution_log", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{
				"step":    fmt.Sprintf("stock_complete:%s", w.CurrentStock),
				"message": fmt.Sprintf("status=%s", ws.Status), "status": "ok",
			}
		}},
	}
}

// BuildReportContent builds evidence-bound report markdown for phase B.
func BuildReportContent(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	evidence := collectReportEvidence(w, code)
	view := buildReportView(ws, evidence)
	lines := []string{
		fmt.Sprintf("# Pre-market Report - %s (%s)", displayStockName(ws, code), code),
		"",
		"## Decision",
		"",
		fmt.Sprintf("- Result: %s", view.Result),
		fmt.Sprintf("- Confidence: %s", view.Confidence),
		fmt.Sprintf("- Suggestion: %s", view.Suggestion),
		fmt.Sprintf("- Reason: %s", view.Reason),
		"",
		"## Key Inputs",
		"",
	}
	for _, item := range view.KeyInputs {
		lines = append(lines, fmt.Sprintf("- %s", item))
	}
	lines = append(lines, "", "## Evidence Refs", "")
	for _, ev := range evidence {
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", ev.ID, ev.Source, ev.Summary))
	}
	if len(evidence) == 0 {
		lines = append(lines, "- No material evidence captured; report should be reviewed before publishing.")
	}
	lines = append(lines, "", "## Data Gaps", "")
	for _, gap := range view.DataGaps {
		lines = append(lines, fmt.Sprintf("- %s", gap))
	}
	if len(view.DataGaps) == 0 {
		lines = append(lines, "- None detected in current workflow inputs.")
	}
	return strings.Join(lines, "\n")
}

// BuildCreateReportArgs builds MCP createPreMarketReport body.
//
// result and confidence are always rule-based (attitude → result, evidence
// count → confidence). reason/suggestion/summary come from the LLM
// synthesizer when configured and successful; otherwise the rule-based view
// is used. The LLM never decides result/confidence.
func BuildCreateReportArgs(w *memory.PreMarketWorking, code string) map[string]any {
	return BuildCreateReportArgsContext(context.Background(), w, code)
}

// BuildCreateReportArgsContext builds MCP createPreMarketReport body using ctx
// for optional LLM synthesis cancellation.
func BuildCreateReportArgsContext(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
	ws := w.Stocks[code]
	var bot memory.BotStock
	for _, b := range w.BotCodes {
		if b.Code == code {
			bot = b
			break
		}
	}
	report := BuildReportContent(w, code)
	attitude := ws.Attitude
	if attitude == "" {
		attitude = "neutral"
	}
	evidence := collectReportEvidence(w, code)
	view := buildReportView(ws, evidence)
	reason := view.Reason
	suggestion := view.Suggestion
	summary := plainSummary(report, 200)
	if synth := SynthesizerFrom(ctx); synth != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		if r, s, sm, err := synth.Synthesize(ctx, ws, evidence, w.MarketContext); err == nil && strings.TrimSpace(r) != "" {
			reason = r
			if strings.TrimSpace(s) != "" {
				suggestion = s
			}
			if strings.TrimSpace(sm) != "" {
				summary = sm
			}
		}
	}
	return map[string]any{
		"code": bot.Code, "stock_name": bot.StockName, "bot_id": bot.BotID,
		"bot_name": bot.BotName, "bot_type": bot.BotType,
		"result": attitudeToResult(attitude), "confidence": view.Confidence,
		"reason": reason, "suggestion": suggestion, "report": report, "summary": summary,
		"evidence_refs": evidenceIDs(evidence),
	}
}

type reportView struct {
	Result     string
	Confidence string
	Suggestion string
	Reason     string
	KeyInputs  []string
	DataGaps   []string
}

func buildReportView(ws memory.StockWorkspace, evidence []memory.EvidenceRef) reportView {
	attitude := ws.Attitude
	if attitude == "" {
		attitude = "neutral"
	}
	result := attitudeToResult(attitude)
	view := reportView{
		Result:     result,
		Confidence: confidenceFor(ws, evidence),
		Suggestion: suggestionFor(result),
		Reason:     reasonFor(ws, evidence),
		KeyInputs: []string{
			fmt.Sprintf("Bot yesterday attitude: %s", attitude),
		},
		DataGaps: dataGaps(ws),
	}
	if ws.WeeklyAnalysisRef != "" {
		view.KeyInputs = append(view.KeyInputs, "Weekly technical analysis captured.")
	}
	if ws.CapitalFlowSummary != "" {
		view.KeyInputs = append(view.KeyInputs, "Capital flow signal: "+ws.CapitalFlowSummary)
	}
	if ws.CapitalDistributionSummary != "" {
		view.KeyInputs = append(view.KeyInputs, "Capital distribution signal: "+ws.CapitalDistributionSummary)
	}
	if ws.StockNewsSummary != "" {
		view.KeyInputs = append(view.KeyInputs, "Stock news signal captured.")
	}
	return view
}

func collectReportEvidence(w *memory.PreMarketWorking, code string) []memory.EvidenceRef {
	items := make([]memory.EvidenceRef, 0, len(w.EvidenceRefs))
	stockPrefix := "stock." + code + "."
	for _, ref := range w.EvidenceRefs {
		if strings.HasPrefix(ref.Source, stockPrefix) || strings.HasPrefix(ref.Source, "market.") {
			items = append(items, ref)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Source == items[j].Source {
			return items[i].ID < items[j].ID
		}
		return items[i].Source < items[j].Source
	})
	return items
}

func confidenceFor(ws memory.StockWorkspace, evidence []memory.EvidenceRef) string {
	if len(evidence) >= 5 && ws.WeeklyAnalysisRef != "" && ws.Attitude != "" {
		return "medium"
	}
	if len(evidence) >= 2 {
		return "low"
	}
	return "review_required"
}

func reasonFor(ws memory.StockWorkspace, evidence []memory.EvidenceRef) string {
	attitude := ws.Attitude
	if attitude == "" {
		attitude = "neutral"
	}
	parts := []string{fmt.Sprintf("bot attitude is %s", attitude)}
	if ws.WeeklyAnalysisRef != "" {
		parts = append(parts, "weekly analysis is available")
	}
	if ws.CapitalFlowSummary != "" {
		parts = append(parts, "capital flow evidence is available")
	}
	if ws.CapitalDistributionSummary != "" {
		parts = append(parts, "capital distribution evidence is available")
	}
	if ws.StockNewsSummary != "" {
		parts = append(parts, "stock news evidence is available")
	}
	if len(evidence) == 0 {
		parts = append(parts, "no evidence refs were captured")
	} else {
		parts = append(parts, fmt.Sprintf("%d evidence ref(s) attached", len(evidence)))
	}
	return strings.Join(parts, "; ")
}

func suggestionFor(result string) string {
	switch result {
	case "long":
		return "watch_long"
	case "short":
		return "reduce_or_avoid"
	default:
		return "hold"
	}
}

func dataGaps(ws memory.StockWorkspace) []string {
	gaps := []string{}
	if ws.WeeklyAnalysisRef == "" {
		gaps = append(gaps, "Missing weekly technical analysis.")
	}
	if ws.StockNewsSummary == "" {
		gaps = append(gaps, "Missing stock-specific news summary.")
	}
	if ws.CapitalFlowSummary == "" {
		gaps = append(gaps, "Missing capital flow summary.")
	}
	if ws.CapitalDistributionSummary == "" {
		gaps = append(gaps, "Missing capital distribution summary.")
	}
	if ws.Attitude == "" {
		gaps = append(gaps, "Missing bot yesterday attitude.")
	}
	return gaps
}

func attitudeToResult(attitude string) string {
	switch attitude {
	case "bullish":
		return "long"
	case "bearish":
		return "short"
	default:
		return "neutral"
	}
}

func evidenceIDs(evidence []memory.EvidenceRef) []string {
	ids := make([]string, 0, len(evidence))
	for _, ev := range evidence {
		ids = append(ids, ev.ID)
	}
	return ids
}

func displayStockName(ws memory.StockWorkspace, code string) string {
	if strings.TrimSpace(ws.StockName) != "" {
		return ws.StockName
	}
	return code
}

func oneLine(s string, n int) string {
	return memory.OneLine(s, n)
}

func plainSummary(markdown string, n int) string {
	lines := strings.Split(markdown, "\n")
	plain := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#- ")
		if line != "" {
			plain = append(plain, line)
		}
	}
	return oneLine(strings.Join(plain, " "), n)
}
