package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// PostMarketPhaseASteps returns post-market prelude (trading day + bot list).
func PostMarketPhaseASteps() []Step {
	return []Step{
		{Name: "check_trading_day", Tool: "check_trading_day", Arguments: map[string]any{"code": tradingDayCode}},
		{Name: "get_report_bot_codes", Tool: "get_report_bot_codes"},
		{Name: "phase_a_complete", Tool: "write_execution_log", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"step": "phase_a_complete", "status": "ok",
				"message": fmt.Sprintf("post_market bots=%d", len(w.BotCodes)),
			}
		}},
	}
}

// PostMarketPerStockSteps returns per-bot post-market analysis steps.
func PostMarketPerStockSteps() []Step {
	return []Step{
		{Name: "list_today_post_market", Tool: "list_today_post_market_reports", ArgFunc: stockReportDateArg},
		{Name: "hourly_price_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlyPricePromptID, "hourly_price")},
		{Name: "hourly_signal_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlySignalPromptID, "hourly_signal")},
		{Name: "hourly_kline_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlyKlinePromptID, "hourly_kline")},
		{Name: "bot_log", Tool: "get_bot_log_by_type", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{"bot_id": ws.BotID, "type": BotLogType(ws.BotType)}
		}},
		{Name: "read_pre_market", Tool: "get_stock_daily_reports", ArgFunc: stockReportDateArg},
		{Name: "current_price", Tool: "get_current_price", ArgFunc: stockCodeArg},
		{Name: "save_local_report", Tool: "save_local_report", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"code": w.CurrentStock, "content": BuildPostMarketReportContent(w, w.CurrentStock),
				"report_type": "postmarket", "report_date": reportDateFor(w, w.CurrentStock),
			}
		}},
		{Name: "create_post_market_report", Tool: "create_post_market_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreatePostMarketReportArgs(ctx, w, w.CurrentStock)
		}},
		{Name: "stock_complete", Tool: "write_execution_log", ArgFunc: stockCompleteArg},
	}
}

// BuildPostMarketReportContent renders post-market markdown.
func BuildPostMarketReportContent(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	sessionDate := reportDateFor(w, code)
	bias := ws.SessionBias
	if bias == "" {
		bias = SessionBiasFromChangePct(ws.ChangePct)
	}
	vs := ws.VsPreMarket
	if vs == "" {
		vs = VsPreMarket(ws.PreMarketResult, bias)
	}
	lines := []string{
		fmt.Sprintf("# 盘后分析报告 - %s (%s)", displayStockName(ws, code), code),
		"",
		fmt.Sprintf("**交易日**: %s", sessionDate),
		"",
		"## 一、今日行情",
		"",
		fmt.Sprintf("- change_pct: %.2f%%", ws.ChangePct),
		fmt.Sprintf("- session_bias: %s", bias),
		"",
	}
	if ws.HourlyPriceAnalysis != "" {
		lines = append(lines, "### 小时级价格分析", ws.HourlyPriceAnalysis, "")
	}
	if ws.HourlySignalAnalysis != "" {
		lines = append(lines, "### 小时级信号分析", ws.HourlySignalAnalysis, "")
	}
	if ws.HourlyKlineAnalysis != "" {
		lines = append(lines, "### 小时级 K 线分析", ws.HourlyKlineAnalysis, "")
	}
	lines = append(lines, "## 二、交易复盘", "", TradeSummaryFromBotLog(ws), "")
	lines = append(lines, "## 三、与盘前对照", "",
		fmt.Sprintf("| 盘前 report_id | %s |", ws.PreMarketReportID),
		fmt.Sprintf("| 盘前 result | %s |", ws.PreMarketResult),
		fmt.Sprintf("| session_bias | %s |", bias),
		fmt.Sprintf("| vs_pre_market | %s |", vs), "")
	lines = append(lines, "## 四、经验与教训", "", ExperienceSummaryDefault(ws, vs), "")
	return strings.Join(lines, "\n")
}

// BuildCreatePostMarketReportArgs builds createPostMarketReport body.
func BuildCreatePostMarketReportArgs(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
	ws := w.Stocks[code]
	sessionDate := reportDateFor(w, code)
	bias := ws.SessionBias
	if bias == "" {
		bias = SessionBiasFromChangePct(ws.ChangePct)
	}
	vs := ws.VsPreMarket
	if vs == "" {
		vs = VsPreMarket(ws.PreMarketResult, bias)
	}
	report := BuildPostMarketReportContent(w, code)
	marketSummary := MarketSummaryFromHourly(ws)
	tradeSummary := TradeSummaryFromBotLog(ws)
	experience := ExperienceSummaryDefault(ws, vs)
	body := map[string]any{
		"code": code, "stock_name": ws.StockName, "session_date": sessionDate,
		"session_bias": bias, "change_pct": ws.ChangePct,
		"trade_summary": tradeSummary, "market_summary": marketSummary,
		"experience_summary": experience, "report": report,
		"summary": plainSummary(report, 200),
		"bot_id": ws.BotID, "bot_name": ws.BotName, "bot_type": ws.BotType,
		"vs_pre_market": vs, "pre_market_report_id": ws.PreMarketReportID,
		"tags": []any{"post_market"},
	}
	_ = ctx
	return body
}
