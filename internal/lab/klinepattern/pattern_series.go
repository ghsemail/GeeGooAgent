package klinepattern

import "context"

// patternHitsSeriesPopulated is true when at least one bar has a TA-Lib pattern hit.
func patternHitsSeriesPopulated(hits []any) bool {
	for _, ph := range hits {
		switch v := ph.(type) {
		case []any:
			if len(v) > 0 {
				return true
			}
		case []map[string]any:
			if len(v) > 0 {
				return true
			}
		}
	}
	return false
}

// barSeriesSameTimes compares aligned OHLCV rows by Shanghai display time (lab vs probe bar shape).
func barSeriesSameTimes(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ra, oka := a[i].(map[string]any)
		rb, okb := b[i].(map[string]any)
		if !oka || !okb {
			return false
		}
		if formatBarTimeAsiaShanghai(ra) != formatBarTimeAsiaShanghai(rb) {
			return false
		}
	}
	return true
}

// ensurePatternHitsOnDailyBars backfills pattern_hits via Signal probe when missing or all-empty.
func ensurePatternHitsOnDailyBars(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	bars []any,
	existing []any,
) []any {
	if run == nil || len(bars) == 0 {
		return existing
	}
	if len(existing) == len(bars) && patternHitsSeriesPopulated(existing) {
		return existing
	}
	probeRaw := map[string]any{}
	EnsurePatternHitsOnProbeRaw(ctx, run, code, "daily", months, bars, probeRaw)
	hits := PatternHitsSeriesFromProbe(probeRaw)
	if len(hits) == 0 {
		return existing
	}
	if len(hits) == len(bars) {
		return hits
	}
	aligned := AlignPatternHitsSeriesToBars(bars, bars, hits)
	if len(aligned) == len(bars) {
		return aligned
	}
	return existing
}