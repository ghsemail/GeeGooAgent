package verdict

// ArbitrateStockPreMarket applies scheme B: LLM suggests, rules finalize.
func ArbitrateStockPreMarket(in StockPreMarketInput) Verdict {
	baseline := baselineStockResult(in.Attitude)
	baselineConf := baselineStockConfidence(in)

	suggested := normalizeResult(in.SuggestedResult)
	suggestedConf := normalizeConfidence(in.SuggestedConfidence)
	market := normalizeResult(in.MarketResult)

	if suggested == "" {
		return Verdict{Result: baseline, Confidence: baselineConf}
	}

	// AI agrees with baseline or stays neutral → keep baseline direction.
	if suggested == "neutral" || suggested == baseline {
		result := baseline
		if baseline == "neutral" && suggested != "neutral" {
			if canUpgradeStock(in, suggested) {
				result = suggested
			}
		}
		aligned := !resultsConflict(baseline, suggested)
		conf := mergeStockConfidence(in, aligned, baselineConf, suggestedConf)
		conf = applyMarketConfidenceBoost(conf, market, result)
		return Verdict{Result: result, Confidence: conf}
	}

	// baseline neutral, AI directional (already handled above if suggested==baseline)
	if baseline == "neutral" {
		if canUpgradeStock(in, suggested) {
			conf := mergeStockConfidence(in, true, baselineConf, suggestedConf)
			conf = applyMarketConfidenceBoost(conf, market, suggested)
			return Verdict{Result: suggested, Confidence: conf}
		}
		return Verdict{
			Result:     "neutral",
			Confidence: minConfidence(baselineConf, ConfidenceLow),
			Note:       "证据不足以在态度中性时采纳 AI 方向建议",
		}
	}

	// Direction conflict: baseline (attitude) wins unless AI and market both disagree.
	if resultsConflict(baseline, suggested) {
		if market != "" && resultsConflict(baseline, market) && normalizeResult(suggested) == market {
			return Verdict{
				Result:     "neutral",
				Confidence: ConfidenceReview,
				Note:       "Bot 态度、AI 建议与市场盘前方向三方冲突，降级为观望",
			}
		}
		conf := downgradeConfidence(mergeStockConfidence(in, false, baselineConf, suggestedConf))
		return Verdict{
			Result:     baseline,
			Confidence: conf,
			Note:       "最终方向采纳 Bot 昨日态度，AI 建议存在反向信号",
		}
	}

	return Verdict{Result: baseline, Confidence: baselineConf}
}

func baselineStockResult(attitude string) string {
	switch attitude {
	case "bullish":
		return "long"
	case "bearish":
		return "short"
	default:
		return "neutral"
	}
}

func baselineStockConfidence(in StockPreMarketInput) string {
	if in.EvidenceCount >= 5 && in.HasWeekly && in.Attitude != "" {
		return ConfidenceMedium
	}
	if in.EvidenceCount >= 2 {
		return ConfidenceLow
	}
	return ConfidenceReview
}

func canUpgradeStock(in StockPreMarketInput, suggested string) bool {
	if in.EvidenceCount < 4 {
		return false
	}
	if !in.HasWeekly {
		return false
	}
	if in.CapitalRequired && !in.HasCapitalFlow && !in.HasCapitalDistribution {
		return false
	}
	if normalizeConfidence(in.SuggestedConfidence) == ConfidenceLow || normalizeConfidence(in.SuggestedConfidence) == "" {
		return false
	}
	market := normalizeResult(in.MarketResult)
	if market != "" && resultsConflict(market, suggested) {
		return false
	}
	return suggested == "long" || suggested == "short"
}

func mergeStockConfidence(in StockPreMarketInput, aligned bool, baselineConf, suggestedConf string) string {
	ruleConf := ruleStockConfidence(in, aligned)
	if suggestedConf == "" {
		return minConfidence(ruleConf, baselineConf)
	}
	return minConfidence(ruleConf, minConfidence(baselineConf, suggestedConf))
}

func ruleStockConfidence(in StockPreMarketInput, aligned bool) string {
	score := 0
	if in.EvidenceCount >= 5 {
		score++
	}
	if in.HasWeekly && (in.HasCapitalFlow || in.HasCapitalDistribution || !in.CapitalRequired) {
		score++
	}
	if aligned {
		score++
	}
	switch {
	case score >= 3:
		return ConfidenceHigh
	case score >= 2:
		return ConfidenceMedium
	case score >= 1:
		return ConfidenceLow
	default:
		return ConfidenceReview
	}
}

func applyMarketConfidenceBoost(conf, marketResult, finalResult string) string {
	market := normalizeResult(marketResult)
	if market == "" || finalResult == "" {
		return conf
	}
	if market == finalResult {
		return upgradeConfidence(conf)
	}
	if resultsConflict(market, finalResult) {
		return downgradeConfidence(conf)
	}
	return conf
}
