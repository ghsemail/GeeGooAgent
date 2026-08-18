package premarket

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/prompts"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/step"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/synthctx"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/templates"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/textutil"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
	"github.com/ghsemail/GeeGooAgent/internal/verdict"
)

// PerStockSteps returns phase B steps for each bot stock.
func PerStockSteps() []step.Step {
	return []step.Step{
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
				"name": ws.StockName, "code": w.CurrentStock, "prompt_id": prompts.IndexPromptID, "period": "weekly", "language": "cn",
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
		{Name: "create_stock_premarket_report", Tool: "create_stock_premarket_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			args := BuildCreateReportArgsContext(ctx, w, w.CurrentStock)
			memory.ApplyPremarketNotifySnapshot(w, w.CurrentStock, args)
			return args
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

// StockReportBundle is the synthesized or rule-based stock pre-market report.
type StockReportBundle struct {
	Report     string
	Result     string
	Confidence string
	Reason     string
	Suggestion string
	Summary    string
	Support    *float64
	Resistance *float64
}

// BuildReportContent builds evidence-bound report markdown for phase B (rule-based draft).
func BuildReportContent(w *memory.PreMarketWorking, code string) string {
	return ensureStockReportBundle(context.Background(), w, code).Report
}

func buildReportContent(w *memory.PreMarketWorking, code string, v *verdict.Verdict, reason, suggestion string) string {
	ws := w.Stocks[code]
	evidence := collectReportEvidence(w, code)
	view := buildReportView(ws, evidence, extractKeyLevels(ws), "")
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
	return buildStockReportDraft(w, code, view)
}

func buildStockReportDraft(w *memory.PreMarketWorking, code string, view reportView) string {
	ws := w.Stocks[code]
	lines := []string{
		"## 市场背景",
		"",
		marketPremarketExcerpt(w),
		"",
		"## 个股新闻",
		"",
		stockNewsSection(ws),
		"",
		"## 资金流向与分布",
		"",
		capitalSection(ws),
		"",
		"## 周线技术分析",
		"",
		weeklyAnalysisSection(ws),
		"",
		"## Bot 盘前态度",
		"",
		botAttitudeSection(ws),
		"",
		"## 综合研判",
		"",
		view.Reason,
		"",
		"### 今日重点关注",
	}
	for _, item := range keyWatchPoints(ws, view, extractKeyLevels(ws)) {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "### 风险提示")
	gaps := view.DataGaps
	if len(gaps) == 0 {
		lines = append(lines, "- 当前输入维度较完整，仍需结合盘中走势验证。")
	} else {
		for _, gap := range gaps {
			lines = append(lines, "- "+gap)
		}
	}
	lines = append(lines,
		"",
		"---",
		"",
		"*报告由 GeeGoo 智能体个股盘前 skill 生成*",
	)
	_ = displayStockName(ws, code)
	return strings.Join(lines, "\n")
}

func ensureStockReportBundle(ctx context.Context, w *memory.PreMarketWorking, code string) StockReportBundle {
	ws := w.Stocks[code]
	evidence := collectReportEvidence(w, code)
	levels := extractKeyLevels(ws)
	view := buildReportView(ws, evidence, levels, w.MarketReportResult)
	draft := buildStockReportDraft(w, code, view)
	bundle := StockReportBundle{
		Report:     draft,
		Result:     view.Result,
		Confidence: view.Confidence,
		Reason:     view.Reason,
		Suggestion: view.Suggestion,
		Summary:    buildStockSummaryOneLiner(ws, view.Result, view.Suggestion),
		Support:    levels.Support,
		Resistance: levels.Resistance,
	}
	synthOut := report.SynthesisResult{}
	stockSynthUsed := false
	if synth := StockPreMarketSynthesizerFrom(ctx); synth != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		if res, err := synth.SynthesizeStockPreMarket(
			ctx, ws, draft, evidence, w.MarketContext, marketPremarketExcerpt(w), templates.LoadStockPremarketTemplate(),
		); err == nil {
			stockSynthUsed = true
			if body := strings.TrimSpace(res.Report); body != "" {
				bundle.Report = body
				bundle.Report = stockfmt.ReplaceMarkdownSection(bundle.Report, "## 个股新闻", stockNewsSection(ws))
				bundle.Report = stockfmt.ReplaceMarkdownSection(bundle.Report, "## 资金流向与分布", capitalSection(ws))
			}
			if v := strings.TrimSpace(res.Reason); v != "" {
				bundle.Reason = v
			}
			if v := strings.TrimSpace(res.Summary); v != "" {
				bundle.Summary = v
			}
			if v := strings.TrimSpace(res.Suggestion); v != "" {
				bundle.Suggestion = v
			}
			synthOut = report.SynthesisResult{
				SuggestedResult:     res.Result,
				SuggestedConfidence: res.Confidence,
				Reason:              res.Reason,
				Suggestion:          res.Suggestion,
				Summary:             res.Summary,
			}
		}
	}
	if !stockSynthUsed {
		if synth := synthctx.From(ctx); synth != nil {
			if ctx == nil {
				ctx = context.Background()
			}
			if res, err := synth.Synthesize(ctx, ws, evidence, w.MarketContext); err == nil && strings.TrimSpace(res.Reason) != "" {
				synthOut = res
				if v := strings.TrimSpace(res.Reason); v != "" {
					bundle.Reason = v
				}
				if v := strings.TrimSpace(res.Suggestion); v != "" {
					bundle.Suggestion = v
				}
				if v := strings.TrimSpace(res.Summary); v != "" {
					bundle.Summary = v
				}
				view.Reason = bundle.Reason
				bundle.Report = buildStockReportDraft(w, code, view)
			}
		}
		if isBoilerplateReason(bundle.Reason) {
			bundle.Reason = buildSubstantiveReason(ws, levels, w.MarketReportResult)
			view.Reason = bundle.Reason
			bundle.Report = buildStockReportDraft(w, code, view)
		}
	}
	final := verdict.ArbitrateStockPreMarket(stockVerdictInput(w, code, ws, evidence, synthOut))
	bundle.Result = final.Result
	bundle.Confidence = final.Confidence
	capDivergent := stockfmt.CapitalFlowDivergent(capitalInterpretationText(ws))
	bundle.Suggestion = finalizeSuggestion(bundle.Result, bundle.Confidence, w.MarketReportResult, capDivergent)
	if strings.TrimSpace(final.Note) != "" {
		bundle.Reason = strings.TrimSpace(bundle.Reason + " " + final.Note)
	}
	bundle.Reason = stockfmt.StripEvidenceRefs(bundle.Reason)
	bundle.Summary = stockfmt.StripEvidenceRefs(bundle.Summary)
	if bundle.Summary == "" || isBoilerplateReason(bundle.Summary) {
		bundle.Summary = buildStockSummaryOneLiner(ws, bundle.Result, bundle.Suggestion)
	}
	bundle.Report = stockfmt.PolishStockNewsInReport(bundle.Report, ws.StockName)
	bundle.Report = stockfmt.PolishStockPremarketMarkdown(bundle.Report)
	return bundle
}

// BuildCreateReportArgs builds MCP createStockPremarketReport body.
func BuildCreateReportArgs(w *memory.PreMarketWorking, code string) map[string]any {
	return BuildCreateReportArgsContext(context.Background(), w, code)
}

// BuildCreateReportArgsContext builds MCP createStockPremarketReport body using ctx
// for optional LLM synthesis cancellation.
func BuildCreateReportArgsContext(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
	var bot memory.BotStock
	for _, b := range w.BotCodes {
		if b.Code == code {
			bot = b
			break
		}
	}
	evidence := collectReportEvidence(w, code)
	bundle := ensureStockReportBundle(ctx, w, code)
	args := map[string]any{
		"code": bot.Code, "stock_name": bot.StockName, "bot_id": bot.BotID,
		"bot_name": bot.BotName, "bot_type": bot.BotType,
		"result": bundle.Result, "confidence": bundle.Confidence,
		"reason": bundle.Reason, "suggestion": bundle.Suggestion, "report": bundle.Report, "summary": bundle.Summary,
		"evidence_refs":              evidenceIDs(evidence),
		"market_premarket_report_id": strings.TrimSpace(w.MarketReportID),
	}
	if bundle.Support != nil {
		args["support"] = *bundle.Support
	}
	if bundle.Resistance != nil {
		args["resistance"] = *bundle.Resistance
	}
	return args
}

func marketPremarketExcerpt(w *memory.PreMarketWorking) string {
	if w == nil {
		return "暂无当日市场盘前摘要。"
	}
	if s := strings.TrimSpace(w.MarketReportSummary); s != "" {
		return s
	}
	if b := strings.TrimSpace(w.MarketReportBody); b != "" {
		return textutil.PlainSummary(b, 400)
	}
	return "暂无当日市场盘前摘要。"
}

func stockNewsSection(ws memory.StockWorkspace) string {
	if s := strings.TrimSpace(ws.StockNewsSummary); s != "" {
		return s
	}
	return "暂无个股新闻。"
}

func capitalSection(ws memory.StockWorkspace) string {
	if s := strings.TrimSpace(ws.CapitalDistributionSummary); s != "" {
		return s
	}
	if s := strings.TrimSpace(ws.CapitalFlowSummary); s != "" {
		return s
	}
	return "暂无资金数据。"
}

func capitalFlowSection(ws memory.StockWorkspace) string {
	return capitalSection(ws)
}

func capitalDistributionSection(ws memory.StockWorkspace) string {
	return ""
}

func weeklyAnalysisSection(ws memory.StockWorkspace) string {
	if s := strings.TrimSpace(ws.WeeklyAnalysisRef); s != "" {
		return s
	}
	return "暂无周线技术分析。"
}

func botAttitudeSection(ws memory.StockWorkspace) string {
	attitude := strings.TrimSpace(ws.Attitude)
	if attitude == "" {
		return "暂无 Bot 昨日态度记录（默认中性）。"
	}
	return fmt.Sprintf("昨日态度为 **%s**。", stockfmt.LocalizeAttitude(attitude))
}

func keyWatchPoints(ws memory.StockWorkspace, _ reportView, levels stockfmt.KeyLevels) []string {
	points := make([]string, 0, 4)
	if ws.WeeklyAnalysisRef != "" {
		points = append(points, keyLevelWatchPoint(levels))
	}
	if ws.CapitalFlowSummary != "" {
		points = append(points, "跟踪主力资金是否延续当前净流入/流出方向。")
	}
	if ws.StockNewsSummary != "" {
		points = append(points, "留意新闻催化兑现后的情绪回落风险。")
	}
	if ws.Attitude != "" && ws.Attitude != "neutral" {
		points = append(points, "结合 Bot 昨日态度，验证开盘走势是否与预期一致。")
	}
	if len(points) == 0 {
		points = append(points, "关注开盘量价与大盘联动。")
	}
	if len(points) > 4 {
		points = points[:4]
	}
	return points
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

func buildReportView(ws memory.StockWorkspace, evidence []memory.EvidenceRef, levels stockfmt.KeyLevels, marketResult string) reportView {
	attitude := ws.Attitude
	if attitude == "" {
		attitude = "neutral"
	}
	result := attitudeToResult(attitude)
	view := reportView{
		Result:     result,
		Confidence: confidenceFor(ws, evidence),
		Suggestion: suggestionFor(result),
		Reason:     buildSubstantiveReason(ws, levels, marketResult),
		KeyInputs: []string{
			fmt.Sprintf("Bot 昨日态度：%s", localizeAttitude(attitude)),
		},
		DataGaps: dataGaps(ws),
	}
	if ws.WeeklyAnalysisRef != "" {
		view.KeyInputs = append(view.KeyInputs, "周线技术分析已纳入研判。")
	}
	if ws.CapitalFlowSummary != "" {
		view.KeyInputs = append(view.KeyInputs, "主力资金流向："+textutil.OneLine(ws.CapitalFlowSummary, 120))
	}
	if ws.CapitalDistributionSummary != "" {
		view.KeyInputs = append(view.KeyInputs, "资金分布结构已纳入研判。")
	}
	if ws.StockNewsSummary != "" {
		view.KeyInputs = append(view.KeyInputs, "个股新闻面已纳入研判。")
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
	return buildSubstantiveReason(ws, extractKeyLevels(ws), "")
}

func suggestionFor(result string) string {
	switch result {
	case "long":
		return "buy"
	case "short":
		return "sell"
	default:
		return "hold"
	}
}

func dataGaps(ws memory.StockWorkspace) []string {
	gaps := []string{}
	if ws.WeeklyAnalysisRef == "" {
		gaps = append(gaps, "缺少周线技术分析。")
	}
	if ws.StockNewsSummary == "" {
		gaps = append(gaps, "缺少个股新闻摘要。")
	}
	if ws.CapitalFlowSummary == "" {
		gaps = append(gaps, "缺少主力资金流向。")
	}
	if ws.CapitalDistributionSummary == "" {
		gaps = append(gaps, "缺少资金分布数据。")
	}
	if ws.Attitude == "" {
		gaps = append(gaps, "缺少 Bot 昨日态度。")
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

func localizeAttitude(attitude string) string {
	switch strings.ToLower(strings.TrimSpace(attitude)) {
	case "bullish", "long":
		return "偏多"
	case "bearish", "short":
		return "偏空"
	default:
		return "中性"
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
