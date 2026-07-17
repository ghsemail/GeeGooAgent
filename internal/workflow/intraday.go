package workflow

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// IntradayInput is caller-provided context for a signal-triggered intraday run.
type IntradayInput struct {
	Code       string
	StockName  string
	BotID      string
	BotName    string
	BotType    string
	Frequency  string
	TradeType  string
	ReportDate string
}

// DefaultIntradayInput returns dry-run friendly defaults.
func DefaultIntradayInput() IntradayInput {
	return IntradayInput{
		Code: "00700.HK", StockName: "腾讯控股",
		BotID: "dry-run-bot-1", BotName: "dry-run-bot", BotType: "DCA",
		Frequency: "5m", TradeType: "信号买入",
	}
}

// IntradayInputFromEnv reads GEEGOO_INTRADAY_* variables.
func IntradayInputFromEnv() IntradayInput {
	in := DefaultIntradayInput()
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_CODE")); v != "" {
		in.Code = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_STOCK_NAME")); v != "" {
		in.StockName = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_BOT_ID")); v != "" {
		in.BotID = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_BOT_NAME")); v != "" {
		in.BotName = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_BOT_TYPE")); v != "" {
		in.BotType = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_FREQUENCY")); v != "" {
		in.Frequency = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_TRADE_TYPE")); v != "" {
		in.TradeType = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_REPORT_DATE")); v != "" {
		in.ReportDate = v
	}
	return in
}

// SeedIntradayWorking prepares a single-stock intraday session.
func SeedIntradayWorking(w *memory.PreMarketWorking, in IntradayInput) {
	if strings.TrimSpace(in.Code) == "" {
		in = DefaultIntradayInput()
	}
	trading := true
	w.IsTradingDay = &trading
	w.Phase = "phase_b"
	bot := memory.BotStock{
		Code: in.Code, StockName: in.StockName,
		BotID: in.BotID, BotName: in.BotName, BotType: in.BotType,
	}
	w.BotCodes = []memory.BotStock{bot}
	w.Stocks[in.Code] = memory.StockWorkspace{
		Code: in.Code, StockName: in.StockName,
		BotID: in.BotID, BotName: in.BotName, BotType: in.BotType,
		Status: "collecting", Frequency: in.Frequency, TradeType: in.TradeType,
		ReportDate: in.ReportDate,
	}
	w.CurrentStock = in.Code
}

// IntradayPhaseASteps is empty; intraday is seeded before phase B.
func IntradayPhaseASteps() []Step { return nil }

// IntradayPerStockSteps returns intraday decision steps for one triggered bot.
func IntradayPerStockSteps() []Step {
	freq := parseFrequencyMinutes(intradayFrequency())
	steps := []Step{
		{Name: "get_position", Tool: "get_position", ArgFunc: stockCodeArg},
		{Name: "read_pre_market", Tool: "get_stock_daily_reports", ArgFunc: stockReportDateArg},
		{Name: "capital_distribution", Tool: "get_capital_distribution", ArgFunc: stockCodeArg},
	}
	if freq > 3 {
		steps = append(steps, Step{
			Name: "hourly_price_analysis", Tool: "get_mcp_analysis",
			ArgFunc: mcpHourlyArg(hourlyPricePromptID, "hourly_price"),
		})
	}
	if freq >= 10 {
		steps = append(steps,
			Step{Name: "hourly_signal_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlySignalPromptID, "hourly_signal")},
			Step{Name: "hourly_kline_analysis", Tool: "get_mcp_analysis", ArgFunc: mcpHourlyArg(hourlyKlinePromptID, "hourly_kline")},
		)
	}
	steps = append(steps,
		Step{Name: "current_price", Tool: "get_current_price", ArgFunc: stockCodeArg},
		Step{Name: "ticker_fallback", Tool: "get_ticker", ArgFunc: stockCodeArg},
		Step{Name: "save_local_report", Tool: "save_local_report", ArgFunc: func(w *memory.PreMarketWorking) map[string]any {
			return map[string]any{
				"code": w.CurrentStock, "content": BuildIntradayReportContent(w, w.CurrentStock),
				"report_type": "intraday", "report_date": reportDateFor(w, w.CurrentStock),
			}
		}},
		Step{Name: "create_intraday_report", Tool: "create_intraday_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreateIntradayReportArgs(ctx, w, w.CurrentStock)
		}},
		Step{Name: "stock_complete", Tool: "write_execution_log", ArgFunc: stockCompleteArg},
	)
	return steps
}

func intradayFrequency() string {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_FREQUENCY")); v != "" {
		return v
	}
	return "5m"
}

func stockCodeArg(w *memory.PreMarketWorking) map[string]any {
	return map[string]any{"code": w.CurrentStock}
}

func stockReportDateArg(w *memory.PreMarketWorking) map[string]any {
	return map[string]any{"code": w.CurrentStock, "report_date": reportDateFor(w, w.CurrentStock)}
}

func mcpHourlyArg(promptID, slot string) func(*memory.PreMarketWorking) map[string]any {
	return func(w *memory.PreMarketWorking) map[string]any {
		ws := w.Stocks[w.CurrentStock]
		return map[string]any{
			"name": ws.StockName, "code": w.CurrentStock,
			"prompt_id": promptID, "period": "hourly", "language": "cn",
			"analysis_slot": slot,
		}
	}
}

func stockCompleteArg(w *memory.PreMarketWorking) map[string]any {
	ws := w.Stocks[w.CurrentStock]
	return map[string]any{
		"step": fmt.Sprintf("stock_complete:%s", w.CurrentStock),
		"message": fmt.Sprintf("status=%s result=%s", ws.Status, ws.IntradayResult),
		"status": "ok",
	}
}

func reportDateFor(w *memory.PreMarketWorking, code string) string {
	if ws, ok := w.Stocks[code]; ok && strings.TrimSpace(ws.ReportDate) != "" {
		return ws.ReportDate
	}
	return todayDate()
}

// BuildIntradayReportContent renders intraday markdown from working state.
func BuildIntradayReportContent(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	result, confidence := ws.IntradayResult, ws.IntradayConfidence
	if result == "" {
		result, confidence = DecideIntraday(ws)
		ws.IntradayResult, ws.IntradayConfidence = result, confidence
	}
	lines := []string{
		fmt.Sprintf("# 盘中交易决策报告 - %s (%s)", displayStockName(ws, code), code),
		"",
		"## 一、决策信息",
		"",
		fmt.Sprintf("| 检查频率 | %s |", ws.Frequency),
		fmt.Sprintf("| 本轮信号 | %s |", ws.TradeType),
		fmt.Sprintf("| 决策结果 | %s |", intradayResultCN(result)),
		fmt.Sprintf("| 置信度 | %s |", confidenceCN(confidence)),
		"",
	}
	if ws.PreMarketResult != "" {
		lines = append(lines, "## 二、盘前报告参考", "",
			fmt.Sprintf("- 盘前判断: %s", preMarketResultCN(ws.PreMarketResult)),
			fmt.Sprintf("- 盘前置信度: %s", confidenceCN(ws.PreMarketConfidence)),
			fmt.Sprintf("- 盘前依据: %s", oneLine(ws.PreMarketReason, 400)),
			"")
	}
	if !isReminderBot(ws.BotType) {
		lines = append(lines, "## 三、当前持仓", "", ws.PositionSummary, "")
	}
	if ws.CapitalDistributionSummary != "" && !isAShareCode(code) {
		lines = append(lines, "## 四、资金分布", "", ws.CapitalDistributionSummary, "")
	}
	if ws.HourlyPriceAnalysis != "" || ws.HourlySignalAnalysis != "" || ws.HourlyKlineAnalysis != "" {
		lines = append(lines, "## 五、小时级分析", "")
		if ws.HourlyPriceAnalysis != "" {
			lines = append(lines, "### 价格", ws.HourlyPriceAnalysis, "")
		}
		if ws.HourlySignalAnalysis != "" {
			lines = append(lines, "### 信号", ws.HourlySignalAnalysis, "")
		}
		if ws.HourlyKlineAnalysis != "" {
			lines = append(lines, "### K线", ws.HourlyKlineAnalysis, "")
		}
	}
	if ws.CurrentPrice > 0 {
		lines = append(lines, "## 六、最新价", "",
			fmt.Sprintf("- 价格来源: %s", ws.PriceSource),
			fmt.Sprintf("- 参考价: %.4f", ws.CurrentPrice), "")
	}
	reason := ws.IntradayReason
	if reason == "" {
		reason = intradayReason(ws, result, confidence)
	}
	lines = append(lines, "## 七、判定依据", "", reason, "")
	return strings.Join(lines, "\n")
}

// BuildCreateIntradayReportArgs builds createIntradayTradeDecisionReport body.
func BuildCreateIntradayReportArgs(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
	ws := w.Stocks[code]
	result, confidence := ws.IntradayResult, ws.IntradayConfidence
	if result == "" {
		result, confidence = DecideIntraday(ws)
	}
	report := BuildIntradayReportContent(w, code)
	reason := ws.IntradayReason
	if reason == "" {
		reason = intradayReason(ws, result, confidence)
	}
	if len([]rune(reason)) < 80 {
		reason = padReason(reason, ws, result)
	}
	body := map[string]any{
		"code": code, "stock_name": ws.StockName,
		"bot_id": ws.BotID, "bot_name": ws.BotName, "bot_type": ws.BotType,
		"result": result, "confidence": confidence, "reason": reason,
		"report": report, "trade_type": ws.TradeType,
		"summary": plainSummary(report, 200), "tags": []any{"intraday"},
	}
	if ws.CurrentPrice > 0 {
		body["price"] = ws.CurrentPrice
	}
	if ws.HasPosition {
		body["position"] = map[string]any{"summary": ws.PositionSummary}
	}
	_ = ctx
	return body
}

func padReason(reason string, ws memory.StockWorkspace, result string) string {
	parts := []string{strings.TrimSpace(reason)}
	if ws.PreMarketResult != "" {
		parts = append(parts, fmt.Sprintf("盘前观点为 %s", ws.PreMarketResult))
	}
	if ws.HourlyPriceAnalysis != "" {
		parts = append(parts, "小时级价格分析已纳入")
	}
	if ws.CurrentPrice > 0 {
		parts = append(parts, fmt.Sprintf("参考价 %.4f", ws.CurrentPrice))
	}
	parts = append(parts, fmt.Sprintf("对本轮信号 %s 的决策为 %s", ws.TradeType, result))
	return strings.Join(parts, "；")
}

func intradayReason(ws memory.StockWorkspace, result, confidence string) string {
	return padReason("", ws, result)
}

func isReminderBot(botType string) bool {
	return strings.Contains(strings.ToLower(botType), "reminder")
}

func isAShareCode(code string) bool {
	return strings.HasSuffix(code, ".SH") || strings.HasSuffix(code, ".SZ")
}

func parseFrequencyMinutes(freq string) int {
	freq = strings.TrimSpace(strings.ToLower(freq))
	if freq == "" {
		return 5
	}
	if strings.HasSuffix(freq, "m") {
		var n int
		fmt.Sscanf(freq, "%d", &n)
		if n > 0 {
			return n
		}
	}
	if strings.HasSuffix(freq, "h") {
		var n int
		fmt.Sscanf(freq, "%d", &n)
		if n > 0 {
			return n * 60
		}
	}
	if strings.HasSuffix(freq, "d") {
		return 1440
	}
	var n int
	if _, err := fmt.Sscanf(freq, "%d", &n); err == nil && n > 0 {
		return n
	}
	return 5
}

func intradayResultCN(result string) string {
	switch result {
	case "buy":
		return "买入"
	case "sell":
		return "卖出"
	default:
		return "观望"
	}
}

func confidenceCN(c string) string {
	switch c {
	case "high":
		return "高"
	case "low":
		return "低"
	default:
		return "中"
	}
}

func preMarketResultCN(r string) string {
	switch r {
	case "long":
		return "看多"
	case "short":
		return "看空"
	default:
		return "中性"
	}
}

func todayDate() string {
	return timeNow().Format("2006-01-02")
}

var timeNow = func() time.Time { return time.Now() }
