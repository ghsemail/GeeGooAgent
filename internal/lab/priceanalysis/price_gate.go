package priceanalysis

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

// priceAnalysisMinRank is the coarsest rank floor for standard four-TF probes and rows.
// 15m diagnose charts align with the 60m upward set (no 5m).
func priceAnalysisMinRank(gateFreq string) int {
	gate := NormalizeFrequency(gateFreq)
	switch gate {
	case "15m":
		return gateFrequencyRank("60m")
	default:
		return gateFrequencyRank(gate)
	}
}

// multiTimeframeSpecsForGate returns 5m/60m/日线/周线 rows to fetch and display:
// signal period plus coarser TFs only (never finer than the diagnose frequency).
func multiTimeframeSpecsForGate(gateFreq string) []struct {
	Label string
	Freq  string
} {
	gate := NormalizeFrequency(gateFreq)
	if gate == "" {
		return multiTimeframeSpecs
	}
	minRank := priceAnalysisMinRank(gateFreq)
	var out []struct {
		Label string
		Freq  string
	}
	for _, spec := range multiTimeframeSpecs {
		if gateFrequencyRank(spec.Freq) >= minRank {
			out = append(out, spec)
		}
	}
	return out
}

// hardGateParticipatingFrequencies lists TFs at or above the diagnose (gate) frequency.
// Example: gate 60m → 60m, daily, weekly (5m is display-only).
func hardGateParticipatingFrequencies(gateFreq string) []string {
	specs := multiTimeframeSpecsForGate(gateFreq)
	out := make([]string, len(specs))
	for i, spec := range specs {
		out[i] = spec.Freq
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

func ohlcvSourceForTimeframe(
	specFreq string,
	chartFreq string,
	barsByFreq map[string][]any,
	lookbackBars []any,
	anchorTime time.Time,
) []any {
	fetched := barsByFreq[specFreq]
	if NormalizeFrequency(chartFreq) != NormalizeFrequency(specFreq) {
		return fetched
	}
	if len(fetched) == 0 {
		return lookbackBars
	}
	if len(lookbackBars) == 0 {
		return fetched
	}
	if len(truncateBarsAtAnchor(lookbackBars, anchorTime)) > len(truncateBarsAtAnchor(fetched, anchorTime)) {
		return lookbackBars
	}
	return fetched
}

func priceFilterResultForFrequency(
	pa PriceAnalysis,
	freq string,
	barsByFreq map[string][]any,
	lookbackBars []any,
	anchorTime time.Time,
	chartFreq string,
) PriceFilterResult {
	freq = NormalizeFrequency(freq)
	src := ohlcvSourceForTimeframe(freq, chartFreq, barsByFreq, lookbackBars, anchorTime)
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
