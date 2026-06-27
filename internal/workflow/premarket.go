package workflow

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

const indexPromptID = "69ec7035b9ccd3d9befc6c23"

var (
	indexEntries = []struct{ Name, Code string }{
		{"道琼斯", "^DJI.US"},
		{"纳斯达克", "^IXIC.US"},
		{"上证指数", "000001.SH"},
		{"深证成指", "399001.SZ"},
		{"恒生指数", "800000.HK"},
	}
	newsMarkets       = []string{"US", "CN", "HK"}
	tradingDayCode    = "00700.HK"
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
			Name: "market_news_" + strings.ToLower(market),
			Tool: "fetch_market_news",
			Arguments: map[string]any{"market": market, "limit": 8},
		})
	}
	steps = append(steps, Step{
		Name: "phase_a_complete", Tool: "write_execution_log",
		ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"step": "phase_a_complete",
				"message": fmt.Sprintf("indices_done=%v market_news_done=%v", w.MarketContext.IndicesDone, w.MarketContext.MarketNewsDone),
				"status": "ok",
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
		{Name: "create_pre_market_report", Tool: "create_pre_market_report", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return BuildCreateReportArgs(w, w.CurrentStock)
		}},
		{Name: "stock_complete", Tool: "write_execution_log", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{
				"step": fmt.Sprintf("stock_complete:%s", w.CurrentStock),
				"message": fmt.Sprintf("status=%s", ws.Status), "status": "ok",
			}
		}},
	}
}

// BuildReportContent builds stub report markdown for phase B.
func BuildReportContent(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	lines := []string{
		fmt.Sprintf("# 盘前报告 — %s (%s)", ws.StockName, code),
		"",
		"## 综合判断",
		"",
		"Go A3 stub report (LLM synthesis deferred to A3+).",
		fmt.Sprintf("- attitude: %s", ws.Attitude),
	}
	return strings.Join(lines, "\n")
}

// BuildCreateReportArgs builds MCP createPreMarketReport body.
func BuildCreateReportArgs(w *memory.PreMarketWorking, code string) map[string]any {
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
	return map[string]any{
		"code": bot.Code, "stock_name": bot.StockName, "bot_id": bot.BotID,
		"bot_name": bot.BotName, "bot_type": bot.BotType,
		"result": attitudeToResult(attitude), "confidence": "medium",
		"reason": fmt.Sprintf("Phase B stub based on attitude=%s", attitude),
		"suggestion": "hold", "report": report, "summary": report[:min(200, len(report))],
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
