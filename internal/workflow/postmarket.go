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
		{Name: "check_trading_day", Tool: "check_trading_day", Arguments: map[string]any{"code": defaultTradingDayCode}},
		{Name: "get_report_bot_codes", Tool: "get_report_bot_codes"},
		{Name: "phase_a_complete", Tool: "write_execution_log", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"step": "phase_a_complete", "status": "ok",
				"message": fmt.Sprintf("postmarket_stock bots=%d", len(w.BotCodes)),
			}
		}},
	}
}

// PostMarketPerStockSteps returns per-bot post-market analysis steps.
func PostMarketPerStockSteps() []Step {
	return []Step{
		{Name: "list_today_postmarket_stock", Tool: "list_today_stock_postmarket_reports", ArgFunc: stockReportDateArg},
		{Name: "hourly_price_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlyPricePromptID, "hourly_price")},
		{Name: "hourly_signal_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlySignalPromptID, "hourly_signal")},
		{Name: "hourly_kline_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlyKlinePromptID, "hourly_kline")},
		{Name: "bot_log", Tool: "get_bot_log_by_type", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{"bot_id": ws.BotID, "type": BotLogType(ws.BotType)}
		}},
		{Name: "read_stock_premarket", Tool: "get_stock_daily_reports", ArgFunc: stockReportDateArg},
		{Name: "current_price", Tool: "get_current_price", ArgFunc: stockCodeArg},
		{Name: "save_local_report", Tool: "save_local_report", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"code": w.CurrentStock, "content": BuildPostMarketReportContent(w, w.CurrentStock),
				"report_type": "postmarket", "report_date": reportDateFor(w, w.CurrentStock),
			}
		}},
		{Name: "create_stock_postmarket_report", Tool: "create_stock_postmarket_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreateStockPostmarketReportArgs(ctx, w, w.CurrentStock)
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
		"## 今日行情",
		"",
		fmt.Sprintf("交易日 %s，涨跌幅 %.2f%%，盘面倾向 %s。", sessionDate, ws.ChangePct, bias),
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
	lines = append(lines, "## 交易复盘", "", TradeSummaryFromBotLog(ws), "")
	lines = append(lines, "## 与盘前对照", "",
		postMarketComparisonNarrative(ws, bias, vs), "")
	lines = append(lines, "## 经验与教训", "", ExperienceSummaryDefault(ws, vs), "",
		"---",
		"",
		"*报告由 GeeGoo 智能体个股盘后 skill 生成*",
	)
	return strings.Join(lines, "\n")
}

func postMarketComparisonNarrative(ws memory.StockWorkspace, bias, vs string) string {
	pre := strings.TrimSpace(ws.PreMarketResult)
	if pre == "" {
		return "当日无盘前报告可对照。"
	}
	parts := []string{
		fmt.Sprintf("盘前判断为 %s", pre),
		fmt.Sprintf("今日盘面倾向 %s", bias),
	}
	if id := strings.TrimSpace(ws.PreMarketReportID); id != "" {
		parts = append(parts, fmt.Sprintf("关联盘前报告 %s", id))
	}
	parts = append(parts, fmt.Sprintf("对照结论：%s", vs))
	return strings.Join(parts, "；") + "。"
}

// BuildCreateStockPostmarketReportArgs builds createStockPostmarketReport body.
func BuildCreateStockPostmarketReportArgs(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
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
		"vs_stock_premarket": vs, "stock_premarket_report_id": ws.PreMarketReportID,
		"tags": []any{"stock_postmarket"},
	}
	_ = ctx
	return body
}
