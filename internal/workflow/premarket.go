package workflow

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/verdict"
)

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

// BuildReportContent builds evidence-bound report markdown for phase B (rule-based draft).
func BuildReportContent(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	evidence := collectReportEvidence(w, code)
	view := buildReportView(ws, evidence)
	return buildReportContent(w, code, &verdict.Verdict{Result: view.Result, Confidence: view.Confidence}, view.Reason, view.Suggestion)
}

func buildReportContent(w *memory.PreMarketWorking, code string, v *verdict.Verdict, reason, suggestion string) string {
	ws := w.Stocks[code]
	evidence := collectReportEvidence(w, code)
	view := buildReportView(ws, evidence)
	if v != nil {
		view.Result = v.Result
		view.Confidence = v.Confidence
	}
	if strings.TrimSpace(reason) != "" {
		view.Reason = reason
	}
	if strings.TrimSpace(suggestion) != "" {
		view.Suggestion = suggestion
	}
	lines := []string{
		fmt.Sprintf("# Pre-market Report - %s (%s)", displayStockName(ws, code), code),
		"",
	}
	if body := strings.TrimSpace(w.MarketReportBody); body != "" {
		lines = append(lines, "## Market Context", "", body, "")
	}
	lines = append(lines,
		"## Decision",
		"",
		fmt.Sprintf("- Result: %s", view.Result),
		fmt.Sprintf("- Confidence: %s", view.Confidence),
		fmt.Sprintf("- Suggestion: %s", view.Suggestion),
		fmt.Sprintf("- Reason: %s", view.Reason),
		"",
		"## Key Inputs",
		"",
	)
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
	evidence := collectReportEvidence(w, code)
	view := buildReportView(ws, evidence)
	synthOut := report.SynthesisResult{}
	if synth := SynthesizerFrom(ctx); synth != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		if res, err := synth.Synthesize(ctx, ws, evidence, w.MarketContext); err == nil && strings.TrimSpace(res.Reason) != "" {
			synthOut = res
		}
	}
	final := verdict.ArbitrateStockPreMarket(stockVerdictInput(w, code, ws, evidence, synthOut))
	reason := view.Reason
	suggestion := view.Suggestion
	summary := ""
	if strings.TrimSpace(synthOut.Reason) != "" {
		reason = synthOut.Reason
	}
	if strings.TrimSpace(synthOut.Suggestion) != "" {
		suggestion = synthOut.Suggestion
	}
	if strings.TrimSpace(synthOut.Summary) != "" {
		summary = synthOut.Summary
	}
	if strings.TrimSpace(final.Note) != "" {
		reason = strings.TrimSpace(reason + " " + final.Note)
	}
	reportBody := buildReportContent(w, code, &final, reason, suggestion)
	if summary == "" {
		summary = plainSummary(reportBody, 200)
	}
	return map[string]any{
		"code": bot.Code, "stock_name": bot.StockName, "bot_id": bot.BotID,
		"bot_name": bot.BotName, "bot_type": bot.BotType,
		"result": final.Result, "confidence": final.Confidence,
		"reason": reason, "suggestion": suggestion, "report": reportBody, "summary": summary,
		"evidence_refs": evidenceIDs(evidence),
		"market_pre_market_report_id": strings.TrimSpace(w.MarketReportID),
	}
}

func stockVerdictInput(
	w *memory.PreMarketWorking,
	code string,
	ws memory.StockWorkspace,
	evidence []memory.EvidenceRef,
	synth report.SynthesisResult,
) verdict.StockPreMarketInput {
	market := NormalizeMarket(w.Market)
	if market == "" {
		market = MarketFromCode(code)
	}
	return verdict.StockPreMarketInput{
		Attitude:               ws.Attitude,
		EvidenceCount:          len(evidence),
		HasWeekly:              ws.WeeklyAnalysisRef != "",
		HasCapitalFlow:         ws.CapitalFlowSummary != "",
		HasCapitalDistribution: ws.CapitalDistributionSummary != "",
		HasStockNews:           ws.StockNewsSummary != "",
		CapitalRequired:        market != MarketCN,
		SuggestedResult:        synth.SuggestedResult,
		SuggestedConfidence:    synth.SuggestedConfidence,
		MarketResult:           w.MarketReportResult,
		MarketConfidence:       w.MarketReportConfidence,
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
