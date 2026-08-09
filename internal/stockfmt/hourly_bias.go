package stockfmt

import "strings"

// InferHourlyBias returns bullish, bearish, or "" when hourly MCP text is neutral or missing.
// Signal analysis is weighted highest, then price, then kline.
func InferHourlyBias(priceAnalysis, signalAnalysis, klineAnalysis string) string {
	for _, raw := range []string{signalAnalysis, priceAnalysis, klineAnalysis} {
		if c := ExtractTaggedConclusion(raw); c != "" {
			if bias := classifyBiasText(c); bias != "" {
				return bias
			}
		}
	}
	combined := strings.TrimSpace(signalAnalysis + priceAnalysis + klineAnalysis)
	if combined == "" {
		return ""
	}
	score := map[string]int{"bullish": 0, "bearish": 0}
	addBiasScore(score, signalAnalysis, 2)
	addBiasScore(score, priceAnalysis, 1)
	addBiasScore(score, klineAnalysis, 1)
	if score["bullish"] > score["bearish"] && score["bullish"] >= 2 {
		return "bullish"
	}
	if score["bearish"] > score["bullish"] && score["bearish"] >= 2 {
		return "bearish"
	}
	return ""
}

func addBiasScore(score map[string]int, raw string, weight int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	bull, bear := biasKeywordScores(raw)
	score["bullish"] += bull * weight
	score["bearish"] += bear * weight
}

func classifyBiasText(s string) string {
	if isNeutralBiasText(s) {
		return ""
	}
	bull, bear := biasKeywordScores(s)
	if bull > bear && bull > 0 {
		return "bullish"
	}
	if bear > bull && bear > 0 {
		return "bearish"
	}
	return ""
}

func isNeutralBiasText(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return true
	}
	neutral := []string{"中性", "neutral", "震荡", "横盘", "观望", "整理"}
	for _, kw := range neutral {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func biasKeywordScores(s string) (bullish, bearish int) {
	lower := strings.ToLower(s)
	for _, kw := range []string{"看多", "偏多", "bullish", "long", "买入", "反弹", "上行", "走强", "突破"} {
		if strings.Contains(lower, kw) {
			bullish++
		}
	}
	for _, kw := range []string{"看空", "偏空", "bearish", "short", "卖出", "下行", "走弱", "承压", "回落", "下跌"} {
		if strings.Contains(lower, kw) {
			bearish++
		}
	}
	return bullish, bearish
}

// HourlyContradictsBuy is true when hourly MCP bias clearly opposes a buy signal.
func HourlyContradictsBuy(priceAnalysis, signalAnalysis, klineAnalysis string) bool {
	return InferHourlyBias(priceAnalysis, signalAnalysis, klineAnalysis) == "bearish"
}

// HourlyContradictsSell is true when hourly MCP bias clearly opposes a sell signal.
func HourlyContradictsSell(priceAnalysis, signalAnalysis, klineAnalysis string) bool {
	return InferHourlyBias(priceAnalysis, signalAnalysis, klineAnalysis) == "bullish"
}
