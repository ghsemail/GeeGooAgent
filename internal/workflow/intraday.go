package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// IntradayInput is caller-provided context for a signal-triggered intraday run.
type IntradayInput struct {
	Code       string
	StockName  string
	BotID      string
	BotName    string
	BotType    string
	Frequency  string
	TradeType      string
	ReportDate     string
	AttitudeSwitch *bool // bot attitude.switch; nil/true reads premarket, false skips
}

// IntradayBundle is the resolved intraday decision and report body.
type IntradayBundle struct {
	Report     string
	Result     string
	Confidence string
	Reason     string
	Summary    string
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
	if sw := intradayAttitudeSwitchFromEnv(); sw != nil {
		in.AttitudeSwitch = sw
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
		ReportDate: in.ReportDate, AttitudeSwitch: intradayAttitudeSwitchEnabled(in),
	}
	w.CurrentStock = in.Code
}

// IntradayPhaseASteps is empty; intraday is seeded before phase B.
func IntradayPhaseASteps() []Step { return nil }

// IntradayPerStockSteps returns intraday decision steps using GEEGOO_INTRADAY_FREQUENCY or 5m default.
func IntradayPerStockSteps() []Step {
	return intradayPerStockSteps(intradayFrequencyFromEnv(), DefaultIntradayInput().BotType, true)
}

func intradayAttitudeSwitchEnabled(in IntradayInput) bool {
	if in.AttitudeSwitch == nil {
		return true
	}
	return *in.AttitudeSwitch
}

func intradayAttitudeSwitchFromEnv() *bool {
	v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_ATTITUDE_SWITCH"))
	if v == "" {
		return nil
	}
	switch strings.ToLower(v) {
	case "0", "false", "off", "no":
		b := false
		return &b
	case "1", "true", "on", "yes":
		b := true
		return &b
	default:
		return nil
	}
}

// IntradayPerStockStepsForWorking builds steps using the seeded stock frequency (API/CLI), then env, then 5m.
func IntradayPerStockStepsForWorking(w *memory.PreMarketWorking) []Step {
	botType := ""
	attitudeSwitch := true
	if w != nil {
		if code := strings.TrimSpace(w.CurrentStock); code != "" {
			ws := w.Stocks[code]
			botType = ws.BotType
			attitudeSwitch = ws.AttitudeSwitch
		}
	}
	return intradayPerStockSteps(intradayFrequencyFromWorking(w), botType, attitudeSwitch)
}

func intradayPerStockSteps(freq, botType string, attitudeSwitch bool) []Step {
	freqMins := parseFrequencyMinutes(freq)
	steps := []Step{
		{Name: "capital_distribution", Tool: "get_capital_distribution", ArgFunc: stockCodeArg},
	}
	if needsIntradayPositionStep(botType) {
		steps = append([]Step{
			{Name: "get_position", Tool: "get_position", ArgFunc: stockCodeArg},
		}, steps...)
	}
	if attitudeSwitch {
		steps = append(steps, Step{
			Name: "read_stock_premarket", Tool: "get_stock_daily_reports", ArgFunc: stockReportDateArg,
		})
	}
	if freqMins >= 10 {
		steps = append(steps, Step{
			Name: "hourly_analysis_bundle", Tool: "get_hourly_analysis_bundle",
			ArgFunc: mcpHourlyBundleArg,
		})
	} else if freqMins > 3 {
		steps = append(steps, Step{
			Name: "hourly_price_analysis", Tool: "get_mcp_analysis",
			ArgFunc: mcpHourlyArg(hourlyPricePromptID, "hourly_price"),
		})
	}
	steps = append(steps,
		Step{Name: "current_price", Tool: "get_current_price", ArgFunc: stockCodeArg},
		Step{Name: "save_local_report", Tool: "save_local_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			bundle := ensureIntradayBundle(ctx, w, w.CurrentStock)
			return map[string]any{
				"code": w.CurrentStock, "content": bundle.Report,
				"report_type": "intraday", "report_date": reportDateFor(w, w.CurrentStock),
			}
		}},
		Step{Name: "create_stock_intraday_report", Tool: "create_stock_intraday_report", ContextArgFunc: func(ctx context.Context, w *memory.PreMarketWorking) map[string]any {
			return BuildCreateIntradayReportArgs(ctx, w, w.CurrentStock)
		}},
		Step{Name: "stock_complete", Tool: "write_execution_log", ArgFunc: stockCompleteArg},
	)
	return steps
}

func needsIntradayPositionStep(botType string) bool {
	return !isReminderBot(botType)
}

func intradayFrequencyFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("GEEGOO_INTRADAY_FREQUENCY")); v != "" {
		return v
	}
	return "5m"
}

func intradayFrequencyFromWorking(w *memory.PreMarketWorking) string {
	if w != nil {
		if code := strings.TrimSpace(w.CurrentStock); code != "" {
			if freq := strings.TrimSpace(w.Stocks[code].Frequency); freq != "" {
				return freq
			}
		}
		for _, ws := range w.Stocks {
			if freq := strings.TrimSpace(ws.Frequency); freq != "" {
				return freq
			}
		}
	}
	return intradayFrequencyFromEnv()
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

func mcpHourlyBundleArg(w *memory.PreMarketWorking) map[string]any {
	ws := w.Stocks[w.CurrentStock]
	return map[string]any{
		"name": ws.StockName, "code": w.CurrentStock, "language": "cn",
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

func intradayBundleArtifactKey(code string) string {
	return "intraday_bundle:" + strings.TrimSpace(code)
}

func ensureIntradayBundle(ctx context.Context, w *memory.PreMarketWorking, code string) IntradayBundle {
	if w == nil {
		return IntradayBundle{}
	}
	if w.Artifacts == nil {
		w.Artifacts = map[string]string{}
	}
	if raw := strings.TrimSpace(w.Artifacts[intradayBundleArtifactKey(code)]); raw != "" {
		var cached IntradayBundle
		if json.Unmarshal([]byte(raw), &cached) == nil && strings.TrimSpace(cached.Result) != "" {
			return cached
		}
	}
	ws := w.Stocks[code]
	draft := buildIntradayDraft(w, code)
	ruleResult, ruleConfidence := DecideIntraday(ws)
	bundle := IntradayBundle{
		Report:     draft,
		Result:     ruleResult,
		Confidence: ruleConfidence,
		Reason:     buildIntradayReason(ws, ruleResult),
		Summary:    stockfmt.IntradayAPISummary(ws.StockName, code, ws.TradeType, ruleResult, ruleConfidence, draft),
	}
	llmUsed := false
	if synth := IntradaySynthesizerFrom(ctx); synth != nil {
		if res, err := synth.SynthesizeIntraday(ctx, ws, draft, ruleResult, ruleConfidence); err == nil {
			llmUsed = true
			if v := strings.TrimSpace(res.Result); v != "" {
				bundle.Result = v
			}
			if v := strings.TrimSpace(res.Confidence); v != "" {
				bundle.Confidence = v
			}
			if v := strings.TrimSpace(res.Summary); v != "" {
				bundle.Summary = v
			}
			if v := strings.TrimSpace(res.Reason); v != "" {
				bundle.Reason = v
			}
		}
	}
	if !llmUsed {
		if len([]rune(bundle.Reason)) < 80 {
			bundle.Reason = buildIntradayReason(ws, bundle.Result)
		}
	}
	bundle.Result = enforceIntradayHardRules(ws, bundle.Result)
	if bundle.Result != ruleResult && llmUsed {
		bundle.Confidence = downgradeIntradayConfidenceAfterOverride(bundle.Confidence)
	}
	bundle.Reason = stockfmt.LocalizeDecisionTerms(bundle.Reason)
	bundle.Summary = stockfmt.LocalizeDecisionTerms(bundle.Summary)
	bundle.Report = assembleIntradayReport(draft, bundle.Reason)

	ws.IntradayResult = bundle.Result
	ws.IntradayConfidence = bundle.Confidence
	ws.IntradayReason = bundle.Reason
	w.Stocks[code] = ws
	if encoded, err := json.Marshal(bundle); err == nil {
		w.Artifacts[intradayBundleArtifactKey(code)] = string(encoded)
	}
	return bundle
}

func enforceIntradayHardRules(ws memory.StockWorkspace, result string) string {
	switch result {
	case "sell":
		if needsIntradayPositionStep(ws.BotType) && !ws.HasPosition {
			return "hold"
		}
	case "buy":
		if ws.AttitudeSwitch && ws.PreMarketResult == "short" && ws.PreMarketConfidence == "high" {
			return "hold"
		}
	}
	return result
}

func downgradeIntradayConfidenceAfterOverride(conf string) string {
	switch conf {
	case "high":
		return "medium"
	default:
		return "low"
	}
}

// BuildIntradayReportContent renders intraday markdown from working state.
func BuildIntradayReportContent(w *memory.PreMarketWorking, code string) string {
	return ensureIntradayBundle(context.Background(), w, code).Report
}

func buildIntradayDraft(w *memory.PreMarketWorking, code string) string {
	ws := w.Stocks[code]
	lines := []string{}
	if ws.AttitudeSwitch && strings.TrimSpace(ws.PreMarketResult) != "" {
		lines = append(lines, "## 盘前报告参考", "",
			fmt.Sprintf("盘前判断 %s，置信度 %s。", preMarketResultCN(ws.PreMarketResult), confidenceCN(ws.PreMarketConfidence)),
			oneLine(ws.PreMarketReason, 400),
			"",
		)
	}
	if needsIntradayPositionStep(ws.BotType) && ws.HasPosition {
		pos := strings.TrimSpace(ws.PositionSummary)
		if pos != "" && pos != "无持仓" {
			lines = append(lines, "## 当前持仓", "", pos, "")
		}
	}
	if ws.CapitalDistributionSummary != "" {
		lines = append(lines, "## 资金分布", "", ws.CapitalDistributionSummary, "")
	}
	if ws.HourlyPriceAnalysis != "" || ws.HourlySignalAnalysis != "" || ws.HourlyKlineAnalysis != "" {
		lines = append(lines, "## 小时级分析", "")
		if ws.HourlyPriceAnalysis != "" {
			lines = append(lines, "### 小时级价格分析", "",
				stockfmt.FormatIntradayHourlySection(ws.HourlyPriceAnalysis, "暂无小时级价格分析。"), "")
		}
		if ws.HourlySignalAnalysis != "" {
			lines = append(lines, "### 小时级信号分析", "",
				stockfmt.FormatIntradaySignalSection(ws.HourlySignalAnalysis), "")
		}
		if ws.HourlyKlineAnalysis != "" {
			lines = append(lines, "### 小时级 K 线分析", "",
				stockfmt.FormatIntradayHourlySection(ws.HourlyKlineAnalysis, "暂无小时级 K 线分析。"), "")
		}
	}
	if ws.CurrentPrice > 0 {
		lines = append(lines, "## 最新价", "",
			fmt.Sprintf("价格来源 %s，参考价 %s。", localizePriceSource(ws.PriceSource), stockfmt.FormatPrice(ws.CurrentPrice)), "")
	}
	return stockfmt.LocalizeDecisionTerms(strings.Join(lines, "\n"))
}

func assembleIntradayReport(draft, reason string) string {
	lines := []string{
		strings.TrimSpace(draft),
		"",
		"## 判定依据",
		"",
		stockfmt.LocalizeDecisionTerms(reason),
		"",
		"---",
		"",
		"*报告由 GeeGoo 智能体个股盘中 skill 生成*",
	}
	return stockfmt.LocalizeDecisionTerms(strings.Join(lines, "\n"))
}

func localizePriceSource(src string) string {
	switch strings.TrimSpace(src) {
	case "get_current_price":
		return "行情快照"
	default:
		if src == "" {
			return "未知"
		}
		return src
	}
}

// BuildCreateIntradayReportArgs builds createStockIntradayReport body.
func BuildCreateIntradayReportArgs(ctx context.Context, w *memory.PreMarketWorking, code string) map[string]any {
	bundle := ensureIntradayBundle(ctx, w, code)
	ws := w.Stocks[code]
	body := map[string]any{
		"code": code, "stock_name": ws.StockName,
		"bot_id": ws.BotID, "bot_name": ws.BotName, "bot_type": ws.BotType,
		"result": bundle.Result, "confidence": bundle.Confidence, "reason": bundle.Reason,
		"report": bundle.Report, "trade_type": ws.TradeType,
		"summary": bundle.Summary,
	}
	if ws.CurrentPrice > 0 {
		body["price"] = ws.CurrentPrice
	}
	if needsIntradayPositionStep(ws.BotType) && ws.HasPosition {
		body["position"] = map[string]any{"summary": ws.PositionSummary}
	}
	return body
}

func buildIntradayReason(ws memory.StockWorkspace, result string) string {
	parts := make([]string, 0, 6)
	if ws.AttitudeSwitch && ws.PreMarketResult != "" {
		parts = append(parts, fmt.Sprintf("盘前观点为%s", preMarketResultCN(ws.PreMarketResult)))
	}
	if ws.CapitalDistributionSummary != "" {
		parts = append(parts, "已参考资金分布")
	}
	if ws.HourlyPriceAnalysis != "" || ws.HourlySignalAnalysis != "" || ws.HourlyKlineAnalysis != "" {
		parts = append(parts, "已结合小时级价格、信号与K线分析")
	}
	if result == "hold" && hasIntradayHourlyData(ws) {
		isBuy := strings.Contains(ws.TradeType, "买") || strings.Contains(strings.ToLower(ws.TradeType), "buy")
		isSell := strings.Contains(ws.TradeType, "卖") || strings.Contains(strings.ToLower(ws.TradeType), "sell")
		if (isBuy && stockfmt.HourlyContradictsBuy(ws.HourlyPriceAnalysis, ws.HourlySignalAnalysis, ws.HourlyKlineAnalysis)) ||
			(isSell && stockfmt.HourlyContradictsSell(ws.HourlyPriceAnalysis, ws.HourlySignalAnalysis, ws.HourlyKlineAnalysis)) {
			parts = append(parts, "小时级分析与信号方向不一致")
		}
	}
	if ws.CurrentPrice > 0 {
		parts = append(parts, fmt.Sprintf("参考价%s", stockfmt.FormatPrice(ws.CurrentPrice)))
	}
	parts = append(parts, fmt.Sprintf("对本轮「%s」信号决策为%s", ws.TradeType, intradayResultCN(result)))
	text := strings.Join(parts, "，") + "。"
	if len([]rune(text)) < 80 {
		text += "综合盘前观点、持仓、资金分布与盘中技术面，建议按结论执行并关注后续量价变化。"
	}
	return stockfmt.LocalizeDecisionTerms(text)
}

func hasIntradayHourlyData(ws memory.StockWorkspace) bool {
	return strings.TrimSpace(ws.HourlyPriceAnalysis+ws.HourlySignalAnalysis+ws.HourlyKlineAnalysis) != ""
}

func isReminderBot(botType string) bool {
	return strings.Contains(strings.ToLower(botType), "reminder")
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
