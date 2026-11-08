package klinepattern

import (
	"context"
	"time"
)

// SupplementDailyMacroMTF re-probes daily pattern series when the initial macro bundle has no bars
// (common for CRYPTO when window end was mis-inferred from 60m chart only).
func SupplementDailyMacroMTF(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	chartBars []any,
	anchorIdx int,
	bundle *KlinePatternMTFBundle,
) {
	if bundle == nil || run == nil || len(chartBars) == 0 {
		return
	}
	key := klinePatternMTFTimeframeKey("daily")
	if ser, ok := bundle.ByFreq[key]; ok && len(ser.Bars) > 0 {
		return
	}
	var asOf *time.Time
	if anchorIdx >= 0 && anchorIdx < len(chartBars) {
		if t := triggerTimeFromBars(chartBars, anchorIdx); !t.IsZero() {
			ts := t
			asOf = &ts
		}
	}
	if asOf == nil {
		_, latest := barSeriesTimeExtent(chartBars)
		if !latest.IsZero() {
			ts := latest
			asOf = &ts
		}
	}
	probeFreq := klinePatternProbeFrequency("daily")
	m := CapMonthsBack(months)
	bars, hits, ok, unsup := fetchKlinePatternProbeSeriesWithRetry(ctx, run, code, probeFreq, m, asOf)
	if !ok || len(bars) == 0 {
		// Retry without as_of in case anchor lies outside the first fetch window.
		bars, hits, ok, unsup = fetchKlinePatternProbeSeriesWithRetry(ctx, run, code, probeFreq, m, nil)
	}
	if len(bars) == 0 {
		return
	}
	bundle.ByFreq[key] = KlinePatternFreqSeries{
		Label:       "日线",
		Freq:        "daily",
		Bars:        bars,
		PatternHits: hits,
		FetchOK:     ok,
		Unsupported: unsup,
	}
}
