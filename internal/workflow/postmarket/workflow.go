package postmarket

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/args"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/decision"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/step"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// PostMarketPhaseASteps returns post-market prelude (trading day + bot list).
func PostMarketPhaseASteps() []step.Step {
	return []step.Step{
		{Name: "check_trading_day", Tool: "check_trading_day", Arguments: map[string]any{"code": args.DefaultTradingDayCode}},
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
func PostMarketPerStockSteps() []step.Step {
	return []step.Step{
		{Name: "list_today_postmarket_stock", Tool: "list_today_stock_postmarket_reports", ArgFunc: args.StockReportDateArg},
		{Name: "hourly_analysis_bundle", Tool: "get_hourly_analysis_bundle", ArgFunc: args.MCPHourlyBundleArg},
		{Name: "bot_log", Tool: "get_bot_log_by_type", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			ws := w.Stocks[w.CurrentStock]
			return map[string]any{"bot_id": ws.BotID, "type": decision.BotLogType(ws.BotType)}
		}},
		{Name: "read_stock_premarket", Tool: "get_stock_daily_reports", ArgFunc: args.PostmarketPremarketLookupArg},
		{Name: "current_price", Tool: "get_current_price", ArgFunc: args.StockCodeArg},
		{Name: "save_local_report", Tool: "save_local_report", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"code": w.CurrentStock, "content": BuildPostMarketReportContent(w, w.CurrentStock),
				"report_type": "postmarket", "report_date": args.ReportDateFor(w, w.CurrentStock),
			}
		}},
		{Name: "create_stock_postmarket_report", Tool: "create_stock_postmarket_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			args := BuildCreateStockPostmarketReportArgs(ctx, w, w.CurrentStock)
			memory.ApplyPostmarketNotifySnapshot(w, w.CurrentStock, args)
			return args
		}},
		{Name: "stock_complete", Tool: "write_execution_log", ArgFunc: args.StockCompleteArg},
	}
}

// BuildPostMarketReportContent renders post-market markdown.
func BuildPostMarketReportContent(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	sessionDate := args.ReportDateFor(w, code)
	bias := ws.SessionBias
	if bias == "" {
		bias = decision.SessionBiasFromChangePct(ws.ChangePct)
	}
	vs := ws.VsPreMarket
	if vs == "" {
		vs = decision.VsPreMarket(ws.PreMarketResult, bias)
	}
	biasLabel := decision.LocalizeSessionBias(bias)
	lines := []string{
		"## 今日行情",
		"",
		"### 涨跌幅与盘面倾向",
		"",
		fmt.Sprintf("交易日 %s，涨跌幅 %.2f%%，盘面倾向 %s。", sessionDate, ws.ChangePct, biasLabel),
		"",
	}
	if ws.HourlyPriceAnalysis != "" {
		lines = append(lines, "### 小时级价格分析", "", stockfmt.FormatEmbeddedHourlyAnalysis(ws.HourlyPriceAnalysis), "")
	}
	if ws.HourlySignalAnalysis != "" {
		lines = append(lines, "### 小时级信号分析", "", stockfmt.FormatHourlySignalAnalysis(ws.HourlySignalAnalysis), "")
	}
	if ws.HourlyKlineAnalysis != "" {
		lines = append(lines, "### 小时级 K 线分析", "", stockfmt.FormatEmbeddedHourlyAnalysis(ws.HourlyKlineAnalysis), "")
	}
	if trade := strings.TrimSpace(decision.TradeSummaryFromBotLog(ws)); trade != "" {
		lines = append(lines, "## 交易复盘", "", trade, "")
	} else {
		lines = append(lines, "## 交易复盘", "", "无", "")
	}
	lines = append(lines, "## 与盘前对照", "",
		postMarketComparisonNarrative(ws, bias, vs), "",
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
		fmt.Sprintf("盘前判断为 %s", localizePreMarketResult(pre)),
		fmt.Sprintf("今日盘面倾向 %s", decision.LocalizeSessionBias(bias)),
		fmt.Sprintf("对照结论：%s", decision.LocalizeVsPreMarket(vs)),
	}
	return strings.Join(parts, "；") + "。"
}

func localizePreMarketResult(result string) string {
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "long", "bullish", "buy":
		return "看多"
	case "short", "bearish", "sell":
		return "看空"
	case "neutral", "hold":
		return "中性"
	default:
		return strings.TrimSpace(result)
	}
}

// BuildCreateStockPostmarketReportArgs builds createStockPostmarketReport body.
func BuildCreateStockPostmarketReportArgs(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
	ws := w.Stocks[code]
	sessionDate := args.ReportDateFor(w, code)
	bias := ws.SessionBias
	if bias == "" {
		bias = decision.SessionBiasFromChangePct(ws.ChangePct)
	}
	vs := ws.VsPreMarket
	if vs == "" {
		vs = decision.VsPreMarket(ws.PreMarketResult, bias)
	}
	report := BuildPostMarketReportContent(w, code)
	marketSummary := decision.MarketSummaryFromHourly(ws)
	tradeSummary := decision.TradeSummaryFromBotLog(ws)
	experience := decision.ExperienceSummaryDefault(ws, vs)
	summary := BuildPostMarketSummaryOneLiner(ws, bias, vs)
	if synth := PostMarketSynthesizerFrom(ctx); synth != nil {
		if res, err := synth.SynthesizePostMarketSummaries(
			ctx, ws, report, bias, vs, marketSummary, tradeSummary, experience,
		); err == nil {
			if v := strings.TrimSpace(res.MarketSummary); v != "" {
				marketSummary = v
			}
			if v := strings.TrimSpace(res.TradeSummary); v != "" {
				tradeSummary = v
			}
			if v := strings.TrimSpace(res.ExperienceSummary); v != "" {
				experience = v
			}
			if v := strings.TrimSpace(res.Summary); v != "" {
				summary = v
			}
		}
	}
	summary = strings.TrimSpace(stockfmt.StripEvidenceRefs(summary))
	if summary == "" {
		summary = BuildPostMarketSummaryOneLiner(ws, bias, vs)
	}
	body := map[string]any{
		"code": code, "stock_name": ws.StockName, "session_date": sessionDate,
		"session_bias": bias, "change_pct": ws.ChangePct,
		"trade_summary": tradeSummary, "market_summary": marketSummary,
		"experience_summary": experience, "report": report,
		"summary": summary,
		"bot_id":  ws.BotID, "bot_name": ws.BotName, "bot_type": ws.BotType,
		"vs_stock_premarket": vs, "stock_premarket_report_id": ws.PreMarketReportID,
		"tags": []any{"stock_postmarket"},
	}
	return body
}
