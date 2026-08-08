package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/verdict"
)

// Market codes for pre-market workflows.
const (
	MarketCN = "CN"
	MarketHK = "HK"
	MarketUS = "US"
)

type marketIndex struct {
	Name string
	Code string
}

var (
	marketIndices = map[string][]marketIndex{
		MarketCN: {
			{Name: "上证指数", Code: "000001.SH"},
			{Name: "深证成指", Code: "399001.SZ"},
		},
		MarketHK: {
			{Name: "恒生指数", Code: "800000.HK"},
		},
		MarketUS: {
			{Name: "道琼斯", Code: "^DJI.US"},
			{Name: "纳斯达克", Code: "^IXIC.US"},
		},
	}
	marketTradingDayCode = map[string]string{
		MarketCN: "000001.SZ",
		MarketHK: "00700.HK",
		MarketUS: "AAPL.US",
	}
	defaultTradingDayCode = "00700.HK"
)

// MarketFromCode infers CN/HK/US from a stock symbol.
func MarketFromCode(code string) string {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(code)), ".")
	if len(parts) < 2 {
		return MarketUS
	}
	switch parts[len(parts)-1] {
	case "HK":
		return MarketHK
	case "US":
		return MarketUS
	case "SH", "SZ", "SS":
		return MarketCN
	default:
		return ""
	}
}

// NormalizeMarket returns CN/HK/US or empty.
func NormalizeMarket(m string) string {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case MarketCN, MarketHK, MarketUS:
		return strings.ToUpper(strings.TrimSpace(m))
	default:
		return ""
	}
}

// MarketPhaseSteps returns phase A for one market pre-market report skill.
func MarketPhaseSteps(market string) []Step {
	market = NormalizeMarket(market)
	checkCode := marketTradingDayCode[market]
	if checkCode == "" {
		checkCode = "00700.HK"
	}
	steps := []Step{
		{Name: "check_trading_day", Tool: "check_trading_day", Arguments: map[string]any{"code": checkCode}},
	}
	for _, idx := range marketIndices[market] {
		steps = append(steps, Step{
			Name: "index_" + idx.Code,
			Tool: "get_mcp_analysis",
			Arguments: map[string]any{
				"name": idx.Name, "code": idx.Code, "prompt_id": indexPromptID, "period": "hourly", "language": "cn",
			},
		})
	}
	steps = append(steps, Step{
		Name:      "market_news_" + strings.ToLower(market),
		Tool:      "fetch_market_news",
		Arguments: map[string]any{"market": market, "limit": 8},
	})
	steps = append(steps,
		Step{Name: "save_local_report", Tool: "save_local_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			bundle := ensureMarketReportBundle(ctx, w, market)
			return map[string]any{
				"code": fmt.Sprintf("market-%s", market), "content": bundle.Report,
				"report_type": "market-premarket",
			}
		}},
		Step{Name: "create_market_pre_market_report", Tool: "create_market_pre_market_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreateMarketReportArgsContext(ctx, w, market)
		}},
		Step{Name: "phase_a_complete", Tool: "write_execution_log", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"step": "phase_a_complete", "message": fmt.Sprintf("market=%s indices=%v", market, w.MarketContext.IndexCodesDone),
				"status": "ok",
			}
		}},
	)
	return steps
}

// StockPhaseASteps returns prelude steps for stock pre-market workflow.
func StockPhaseASteps(market string) []Step {
	market = NormalizeMarket(market)
	return []Step{
		{Name: "get_market_pre_market_report", Tool: "get_market_pre_market_report", Arguments: map[string]any{"market": market}},
		{Name: "get_report_bot_codes", Tool: "get_report_bot_codes"},
		Step{Name: "phase_a_complete", Tool: "write_execution_log", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"step": "phase_a_complete", "message": fmt.Sprintf("market=%s stocks=%d", market, len(w.BotCodes)),
				"status": "ok",
			}
		}},
	}
}

// MarketReportBundle is the resolved market report payload for save/API.
type MarketReportBundle struct {
	Report     string
	Summary    string
	Result     string
	Confidence string
}

// BuildMarketReportContent renders the rule-based draft used as LLM input/fallback.
func BuildMarketReportContent(w *memory.PreMarketWorking, market string) string {
	return buildMarketReportDraft(w, market)
}

func buildMarketReportDraft(w *memory.PreMarketWorking, market string) string {
	market = NormalizeMarket(market)
	lines := []string{
		fmt.Sprintf("# %s 市场盘前报告", marketLabel(market)),
		"",
		fmt.Sprintf("**市场**: %s", market),
		"",
		"## 一、指数走势（hourly MCP）",
		"",
	}
	indexBlocks := marketIndexBlocks(market, w.MarketContext.IndexAnalysisRefs)
	if len(indexBlocks) == 0 {
		lines = append(lines, "- 暂无指数分析数据。")
	} else {
		lines = append(lines, indexBlocks...)
	}
	lines = append(lines, "", "## 二、市场新闻摘要", "")
	newsText := strings.TrimSpace(w.MarketContext.MarketNews[market])
	if newsText != "" {
		lines = append(lines, newsText)
	} else {
		lines = append(lines, "- 暂无市场新闻。")
	}
	summary := oneLine(plainSummary(strings.Join(indexBlocks, "\n")+"\n"+newsText, 200), 200)
	if summary == "" {
		summary = "暂无"
	}
	fallback := fallbackMarketJudgement(w, market)
	lines = append(lines,
		"",
		"## 三、市场综合预判",
		"",
		"| 字段 | 值 |",
		"|------|-----|",
		fmt.Sprintf("| 情绪 (result) | %s |", fallback.Result),
		fmt.Sprintf("| 置信度 (confidence) | %s |", fallback.Confidence),
		fmt.Sprintf("| 摘要 (summary) | %s |", summary),
		"",
		"---",
		"",
		"**报告生成**: geegoo-agent · skill `pre_market`",
	)
	return strings.Join(lines, "\n")
}

func ensureMarketReportBundle(ctx context.Context, w *memory.PreMarketWorking, market string) MarketReportBundle {
	market = NormalizeMarket(market)
	if w.MarketReportSynthesized && strings.TrimSpace(w.MarketReportBody) != "" {
		return MarketReportBundle{
			Report:     w.MarketReportBody,
			Summary:    nonEmptyMarket(w.MarketReportSummary, plainSummary(w.MarketReportBody, 200)),
			Result:     nonEmptyMarket(w.MarketReportResult, "neutral"),
			Confidence: nonEmptyMarket(w.MarketReportConfidence, "medium"),
		}
	}
	draft := buildMarketReportDraft(w, market)
	bundle := MarketReportBundle{
		Report:     draft,
		Summary:    plainSummary(draft, 200),
		Result:     fallbackMarketJudgement(w, market).Result,
		Confidence: fallbackMarketJudgement(w, market).Confidence,
	}
	if synth := MarketSynthesizerFrom(ctx); synth != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		evidence := collectMarketEvidence(w, market)
		if res, err := synth.SynthesizeMarket(ctx, market, draft, w.MarketContext, evidence, loadMarketReportTemplate()); err == nil {
			if body := strings.TrimSpace(res.Report); body != "" {
				bundle.Report = body
			}
			if v := strings.TrimSpace(res.Summary); v != "" {
				bundle.Summary = v
			}
			final := verdict.ArbitrateMarketPreMarket(verdict.MarketPreMarketInput{
				IndicesDone:         w.MarketContext.IndicesDone,
				MarketNewsDone:      w.MarketContext.MarketNewsDone,
				EvidenceCount:       len(evidence),
				SuggestedResult:     res.Result,
				SuggestedConfidence: res.Confidence,
			})
			bundle.Result = final.Result
			bundle.Confidence = final.Confidence
			w.MarketReportSynthesized = true
		}
	}
	w.MarketReportBody = bundle.Report
	w.MarketReportSummary = bundle.Summary
	w.MarketReportResult = bundle.Result
	w.MarketReportConfidence = bundle.Confidence
	return bundle
}

// BuildCreateMarketReportArgs builds MCP createMarketPreMarketReport body.
func BuildCreateMarketReportArgs(w *memory.PreMarketWorking, market string) map[string]any {
	return BuildCreateMarketReportArgsContext(context.Background(), w, market)
}

// BuildCreateMarketReportArgsContext builds MCP createMarketPreMarketReport body with optional LLM synthesis.
func BuildCreateMarketReportArgsContext(ctx context.Context, w *memory.PreMarketWorking, market string) map[string]any {
	bundle := ensureMarketReportBundle(ctx, w, market)
	market = NormalizeMarket(market)
	out := map[string]any{
		"market":  market,
		"report":  bundle.Report,
		"summary": bundle.Summary,
	}
	if bundle.Result != "" {
		out["result"] = bundle.Result
	}
	if bundle.Confidence != "" {
		out["confidence"] = bundle.Confidence
	}
	return out
}

func collectMarketEvidence(w *memory.PreMarketWorking, market string) []memory.EvidenceRef {
	if w == nil {
		return nil
	}
	market = strings.ToUpper(strings.TrimSpace(market))
	out := make([]memory.EvidenceRef, 0, len(w.EvidenceRefs))
	for _, ref := range w.EvidenceRefs {
		if strings.HasPrefix(ref.Source, "market.") || strings.Contains(ref.Source, "."+market+".") {
			out = append(out, ref)
			continue
		}
		for _, idx := range marketIndices[market] {
			if strings.Contains(ref.Source, idx.Code) || strings.Contains(ref.Summary, idx.Code) {
				out = append(out, ref)
				break
			}
		}
	}
	return out
}

func fallbackMarketJudgement(w *memory.PreMarketWorking, market string) MarketReportBundle {
	result := "neutral"
	confidence := "low"
	if w != nil && w.MarketContext.IndicesDone {
		confidence = "medium"
	}
	if w != nil && w.MarketContext.IndicesDone && w.MarketContext.MarketNewsDone {
		if news := strings.TrimSpace(w.MarketContext.MarketNews[NormalizeMarket(market)]); news != "" {
			confidence = "high"
		}
	}
	return MarketReportBundle{Result: result, Confidence: confidence}
}

func loadMarketReportTemplate() string {
	const rel = "skills/pre_market/template.md"
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 8; i++ {
		path := filepath.Join(dir, rel)
		if raw, err := os.ReadFile(path); err == nil {
			return string(raw)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func nonEmptyMarket(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func marketLabel(market string) string {
	switch NormalizeMarket(market) {
	case MarketCN:
		return "A股"
	case MarketHK:
		return "港股"
	case MarketUS:
		return "美股"
	default:
		return market
	}
}

func marketIndexBlocks(market string, refs map[string]string) []string {
	if len(refs) == 0 {
		return nil
	}
	type pair struct{ title, code string }
	var order []pair
	for _, idx := range marketIndices[NormalizeMarket(market)] {
		order = append(order, pair{title: idx.Name, code: idx.Code})
	}
	out := []string{}
	for _, p := range order {
		summary, ok := refs[p.code]
		if !ok {
			continue
		}
		out = append(out,
			fmt.Sprintf("### %s (%s)", p.title, p.code),
			"| 指标 | 值 |",
			"|------|-----|",
			fmt.Sprintf("| 分析结论 | %s |", oneLine(summary, 600)),
			"",
		)
	}
	for code, summary := range refs {
		found := false
		for _, p := range order {
			if p.code == code {
				found = true
				break
			}
		}
		if found {
			continue
		}
		out = append(out,
			fmt.Sprintf("### %s", code),
			fmt.Sprintf("- %s", oneLine(summary, 600)),
			"",
		)
	}
	return out
}

// ExpectedIndexCount returns how many indices a market workflow should collect.
func ExpectedIndexCount(market string) int {
	return len(marketIndices[NormalizeMarket(market)])
}

// SeedMarketWorking sets market scope on working memory before a run.
func SeedMarketWorking(w *memory.PreMarketWorking, market string) {
	if w == nil {
		return
	}
	w.Market = NormalizeMarket(market)
}
