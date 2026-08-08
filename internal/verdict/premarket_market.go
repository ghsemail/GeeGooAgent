package verdict

// ArbitrateMarketPreMarket finalizes market-level premarket_market result/confidence.
// Baseline is neutral with confidence from data completeness; LLM may upgrade
// when indices and news are captured.
func ArbitrateMarketPreMarket(in MarketPreMarketInput) Verdict {
	baselineConf := baselineMarketConfidence(in)
	suggested := normalizeResult(in.SuggestedResult)
	suggestedConf := normalizeConfidence(in.SuggestedConfidence)

	if suggested == "" {
		return Verdict{Result: "neutral", Confidence: baselineConf}
	}

	if canUpgradeMarket(in, suggested) {
		conf := minConfidence(baselineMarketConfidence(in), suggestedConf)
		if conf == "" {
			conf = baselineConf
		}
		return Verdict{Result: suggested, Confidence: conf}
	}

	if suggested == "neutral" {
		return Verdict{Result: "neutral", Confidence: minConfidence(baselineConf, suggestedConf)}
	}

	return Verdict{
		Result:     "neutral",
		Confidence: minConfidence(baselineConf, ConfidenceLow),
		Note:       "市场数据不完整，不采纳 AI 方向建议",
	}
}

func baselineMarketConfidence(in MarketPreMarketInput) string {
	if in.IndicesDone && in.MarketNewsDone {
		return ConfidenceHigh
	}
	if in.IndicesDone || in.MarketNewsDone {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

func canUpgradeMarket(in MarketPreMarketInput, suggested string) bool {
	if suggested == "neutral" {
		return true
	}
	if !in.IndicesDone || !in.MarketNewsDone {
		return false
	}
	if in.EvidenceCount < 2 && !in.IndicesDone {
		return false
	}
	conf := normalizeConfidence(in.SuggestedConfidence)
	return conf == ConfidenceMedium || conf == ConfidenceHigh
}
