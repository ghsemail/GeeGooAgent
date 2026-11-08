package klinepattern

// AlignPatternHitsSeriesToBars maps probe pattern_hits onto chart bars by bar time (after months_back clamp).
func AlignPatternHitsSeriesToBars(chartBars, probeBars []any, hits []any) []any {
	if len(chartBars) == 0 || len(probeBars) == 0 || len(hits) == 0 {
		return nil
	}
	if len(chartBars) == len(hits) && len(chartBars) == len(probeBars) {
		return hits
	}
	out := make([]any, len(chartBars))
	for i := range chartBars {
		trigger := triggerTimeFromBars(chartBars, i)
		if trigger.IsZero() {
			continue
		}
		j := alignBarIndexAtOrBefore(probeBars, trigger)
		if j < 0 || j >= len(hits) {
			continue
		}
		out[i] = hits[j]
	}
	return out
}
