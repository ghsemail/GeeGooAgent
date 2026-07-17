package workflow

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// DecideIntraday applies geegoo intraday decision rules (Step 5.5).
func DecideIntraday(ws memory.StockWorkspace) (result, confidence string) {
	result = "hold"
	confidence = "medium"
	isBuy := isBuyTradeType(ws.TradeType)
	isSell := isSellTradeType(ws.TradeType)
	reminder := isReminderBot(ws.BotType)

	if isSell && !reminder && !ws.HasPosition {
		return "hold", downgradeConfidence(confidence, ws)
	}
	if isBuy {
		if ws.PreMarketResult == "short" && ws.PreMarketConfidence == "high" {
			return "hold", downgradeConfidence(confidence, ws)
		}
		if isBuyAligned(ws.PreMarketResult) {
			return "buy", confidenceForIntraday(ws)
		}
		return "hold", downgradeConfidence(confidence, ws)
	}
	if isSell {
		if ws.PreMarketResult == "long" && ws.PreMarketConfidence == "high" {
			return "hold", downgradeConfidence(confidence, ws)
		}
		if isSellAligned(ws.PreMarketResult) || reminder {
			return "sell", confidenceForIntraday(ws)
		}
		return "hold", downgradeConfidence(confidence, ws)
	}
	return result, downgradeConfidence(confidence, ws)
}

func isBuyTradeType(tradeType string) bool {
	t := strings.ToLower(tradeType)
	return strings.Contains(tradeType, "买") || strings.Contains(t, "buy")
}

func isSellTradeType(tradeType string) bool {
	t := strings.ToLower(tradeType)
	return strings.Contains(tradeType, "卖") || strings.Contains(t, "sell")
}

func isBuyAligned(preResult string) bool {
	return preResult == "" || preResult == "long" || preResult == "neutral"
}

func isSellAligned(preResult string) bool {
	return preResult == "" || preResult == "short" || preResult == "neutral"
}

func confidenceForIntraday(ws memory.StockWorkspace) string {
	score := 0
	if ws.PreMarketResult != "" {
		score++
	}
	if ws.HourlyPriceAnalysis != "" {
		score++
	}
	if ws.CurrentPrice > 0 {
		score++
	}
	if ws.CapitalDistributionSummary != "" {
		score++
	}
	if score >= 3 {
		return "high"
	}
	if score >= 2 {
		return "medium"
	}
	return "low"
}

func downgradeConfidence(base string, ws memory.StockWorkspace) string {
	if ws.PreMarketResult == "" || ws.CurrentPrice <= 0 {
		if base == "high" {
			return "medium"
		}
		return "low"
	}
	return base
}

// SessionBiasFromChangePct maps change_pct to session_bias per geegoo post-market rules.
func SessionBiasFromChangePct(pct float64) string {
	if pct > 1 {
		return "bullish"
	}
	if pct < -1 {
		return "bearish"
	}
	return "neutral"
}

// VsPreMarket compares pre_market result with session_bias.
func VsPreMarket(preResult, sessionBias string) string {
	if preResult == "" {
		return "na"
	}
	pre := strings.ToLower(strings.TrimSpace(preResult))
	bias := strings.ToLower(strings.TrimSpace(sessionBias))
	aligned := (pre == "long" && bias == "bullish") ||
		(pre == "short" && bias == "bearish") ||
		(pre == "neutral" && bias == "neutral")
	if aligned {
		return "aligned"
	}
	contradicted := (pre == "long" && bias == "bearish") || (pre == "short" && bias == "bullish")
	if contradicted {
		return "contradicted"
	}
	return "partial"
}

// BotLogType maps bot_type to getBotLogByType type parameter.
func BotLogType(botType string) string {
	bt := strings.ToUpper(botType)
	if strings.Contains(bt, "GRID") {
		return "GRID"
	}
	return "DCA"
}

// MarketSummaryFromHourly builds a minimum market_summary paragraph.
func MarketSummaryFromHourly(ws memory.StockWorkspace) string {
	parts := []string{}
	if ws.ChangePct != 0 {
		parts = append(parts, fmt.Sprintf("收盘涨跌约 %.2f%%", ws.ChangePct))
	}
	if ws.HourlyPriceAnalysis != "" {
		parts = append(parts, oneLine(ws.HourlyPriceAnalysis, 120))
	}
	if ws.HourlySignalAnalysis != "" {
		parts = append(parts, oneLine(ws.HourlySignalAnalysis, 80))
	}
	if len(parts) == 0 {
		return "今日行情数据不完整，盘面倾向依据有限，建议结合盘前报告与持仓日志复核。"
	}
	text := strings.Join(parts, "；")
	if len([]rune(text)) < 80 {
		text += "。量能与关键价位需结合小时级分析与 Bot 日志综合判断，避免单指标过度解读。"
	}
	return text
}

// TradeSummaryFromBotLog builds trade_summary from bot log snapshot.
func TradeSummaryFromBotLog(ws memory.StockWorkspace) string {
	if strings.TrimSpace(ws.BotLogSummary) != "" {
		return oneLine(ws.BotLogSummary, 400)
	}
	if ws.HasPosition {
		return fmt.Sprintf("当前持仓：%s", ws.PositionSummary)
	}
	return "今日 Bot 日志未返回有效成交记录，持仓状态以账户同步为准。"
}

// ExperienceSummaryDefault builds a post-market experience paragraph.
func ExperienceSummaryDefault(ws memory.StockWorkspace, vs string) string {
	return fmt.Sprintf(
		"今日盘面 session_bias=%s，与盘前对照为 %s。复盘时应优先核对盘前观点与盘中实际走势是否一致，"+
			"并记录 Bot 在 %s 频率下的信号触发与执行偏差，便于后续调整 attitude 开关或止盈止损参数。",
		ws.SessionBias, vs, ws.BotType,
	)
}
