package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
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
			return map[string]any{
				"code": fmt.Sprintf("market-%s", market), "content": BuildMarketReportContent(w, market),
				"report_type": "market-premarket",
			}
		}},
		Step{Name: "create_market_pre_market_report", Tool: "create_market_pre_market_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreateMarketReportArgs(w, market)
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

// BuildMarketReportContent renders global market pre-market markdown.
func BuildMarketReportContent(w *memory.PreMarketWorking, market string) string {
	market = NormalizeMarket(market)
	lines := []string{
		fmt.Sprintf("# Market Pre-market Report — %s", market),
		"",
		"## Indices",
		"",
	}
	if len(w.MarketContext.IndexAnalysisRefs) == 0 {
		lines = append(lines, "- No index analysis captured.")
	}
	for code, summary := range w.MarketContext.IndexAnalysisRefs {
		lines = append(lines, fmt.Sprintf("- **%s**: %s", code, oneLine(summary, 400)))
	}
	lines = append(lines, "", "## Market News", "")
	if text := strings.TrimSpace(w.MarketContext.MarketNews[market]); text != "" {
		lines = append(lines, text)
	} else {
		lines = append(lines, "- No market news captured.")
	}
	return strings.Join(lines, "\n")
}

// BuildCreateMarketReportArgs builds MCP createMarketPreMarketReport body.
func BuildCreateMarketReportArgs(w *memory.PreMarketWorking, market string) map[string]any {
	market = NormalizeMarket(market)
	report := BuildMarketReportContent(w, market)
	return map[string]any{
		"market":  market,
		"report":  report,
		"summary": plainSummary(report, 200),
	}
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
