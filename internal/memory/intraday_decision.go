package memory

import "strings"

// DecideIntraday applies geegoo intraday decision rules (Step 5.5).
func DecideIntraday(ws StockWorkspace) (result, confidence string) {
	result = "hold"
	confidence = "medium"
	isBuy := isBuyTradeType(ws.TradeType)
	isSell := isSellTradeType(ws.TradeType)
	reminder := isReminderBotType(ws.BotType)

	if isSell && !reminder && !ws.HasPosition {
		return "hold", downgradeIntradayConfidence(confidence, ws)
	}
	if isBuy {
		if ws.PreMarketResult == "short" && ws.PreMarketConfidence == "high" {
			return "hold", downgradeIntradayConfidence(confidence, ws)
		}
		if isBuyAligned(ws.PreMarketResult) {
			return "buy", confidenceForIntraday(ws)
		}
		return "hold", downgradeIntradayConfidence(confidence, ws)
	}
	if isSell {
		if ws.PreMarketResult == "long" && ws.PreMarketConfidence == "high" {
			return "hold", downgradeIntradayConfidence(confidence, ws)
		}
		if isSellAligned(ws.PreMarketResult) || reminder {
			return "sell", confidenceForIntraday(ws)
		}
		return "hold", downgradeIntradayConfidence(confidence, ws)
	}
	return result, downgradeIntradayConfidence(confidence, ws)
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

func isReminderBotType(botType string) bool {
	return strings.Contains(strings.ToLower(botType), "reminder")
}

func confidenceForIntraday(ws StockWorkspace) string {
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

func downgradeIntradayConfidence(base string, ws StockWorkspace) string {
	if ws.PreMarketResult == "" || ws.CurrentPrice <= 0 {
		if base == "high" {
			return "medium"
		}
		return "low"
	}
	return base
}
