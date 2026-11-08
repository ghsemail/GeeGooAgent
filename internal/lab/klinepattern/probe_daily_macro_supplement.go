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

// HydrateDailyMacroMTFFromBars attaches lab-fetched daily OHLCV to the bundle and backfills pattern_hits.
func HydrateDailyMacroMTFFromBars(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	dailyBars []any,
	bundle *KlinePatternMTFBundle,
) {
	if bundle == nil || run == nil || len(dailyBars) == 0 {
		return
	}
	if bundle.ByFreq == nil {
		bundle.ByFreq = map[string]KlinePatternFreqSeries{}
	}
	key := klinePatternMTFTimeframeKey("daily")
	ser := bundle.ByFreq[key]
	if len(ser.Bars) > 0 && len(ser.PatternHits) == len(ser.Bars) {
		return
	}
	probeRaw := map[string]any{}
	EnsurePatternHitsOnProbeRaw(ctx, run, code, "daily", months, dailyBars, probeRaw)
	hits := PatternHitsSeriesFromProbe(probeRaw)
	if len(hits) == 0 {
		hits = ser.PatternHits
	}
	bundle.SignalFreq = "daily"
	bundle.ByFreq[key] = KlinePatternFreqSeries{
		Label:       "日线",
		Freq:        "daily",
		Bars:        dailyBars,
		PatternHits: hits,
		FetchOK:     true,
	}
}
