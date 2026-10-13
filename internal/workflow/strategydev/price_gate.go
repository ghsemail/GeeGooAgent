package strategydev

import "time"

// gateFrequencyRank orders timeframes for hard-gate participation (higher = coarser).
func gateFrequencyRank(freq string) int {
	switch NormalizeFrequency(freq) {
	case "5m":
		return 10
	case "15m":
		return 20
	case "60m":
		return 30
	case "daily":
		return 40
	case "weekly", "1wk":
		return 50
	default:
		return 30
	}
}

// hardGateParticipatingFrequencies lists TFs at or above the diagnose (gate) frequency.
// Example: gate 60m → 60m, daily, weekly (5m is display-only).
func hardGateParticipatingFrequencies(gateFreq string) []string {
	gate := NormalizeFrequency(gateFreq)
	minRank := gateFrequencyRank(gate)
	var out []string
	for _, spec := range multiTimeframeSpecs {
		if gateFrequencyRank(spec.Freq) >= minRank {
			out = append(out, spec.Freq)
		}
	}
	return out
}

func rowMatchesGateFrequency(r PriceAnalysisRow, freq string) bool {
	label := timeframeDisplayLabel(freq)
	return r.Timeframe == label || r.Timeframe == freq
}

func priceFilterResultFromRowConclusion(r PriceAnalysisRow, freq string) PriceFilterResult {
	pf := PriceFilterResult{
		Regime:    r.Conclusion,
		Frequency: NormalizeFrequency(freq),
		BarCount:  0,
	}
	pf.AllowLong, pf.AllowShort = allowDirectionsFromFilter(pf)
	return pf
}

func priceFilterResultForFrequency(
	pa PriceAnalysis,
	freq string,
	barsByFreq map[string][]any,
	chartBars []any,
	anchorTime time.Time,
	chartFreq string,
) PriceFilterResult {
	freq = NormalizeFrequency(freq)
	src := barsByFreq[freq]
	if len(src) == 0 && freq == NormalizeFrequency(chartFreq) {
		src = chartBars
	}
	trunc := truncateBarsAtAnchor(src, anchorTime)
	if len(trunc) == 0 {
		for _, r := range pa.Rows {
			if rowMatchesGateFrequency(r, freq) {
				return priceFilterResultFromRowConclusion(r, freq)
			}
		}
		pf := PriceFilterResult{Regime: RegimeUnclear, Frequency: freq}
		pf.AllowLong, pf.AllowShort = allowDirectionsFromFilter(pf)
		return pf
	}
	return AnalyzePriceFilter(tailBarsForPriceAnalysis(trunc, freq), freq)
}

// shouldInterceptTrigger applies episode hard-gate rules per trigger side.
func shouldInterceptTrigger(side string, pf PriceFilterResult) bool {
	if side == "buy" {
		return !pf.AllowLong
	}
	// Sell: block short only when the gate (diagnose) timeframe is in an uptrend.
	return pf.Regime == RegimeUp
}
