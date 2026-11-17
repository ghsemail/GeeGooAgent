package klinepattern

import (
	"context"
	"time"
)

// SupplementDailyMacroMTF re-probes daily pattern series when the initial macro bundle has no bars
// (common for CRYPTO when window end was mis-inferred from 60m chart only).
// Deprecated for macro monitor: use SupplementDailyMacroMTFAtAsOf with session as_of only.
func SupplementDailyMacroMTF(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	chartBars []any,
	anchorIdx int,
	bundle *KlinePatternMTFBundle,
) {
	SupplementDailyMacroMTFAtAsOf(ctx, run, code, months, nil, bundle, true)
	_ = anchorIdx
	_ = chartBars
}

// SupplementDailyMacroMTFAtAsOf fetches daily pattern series anchored at session asOf (no chart buy idx).
func SupplementDailyMacroMTFAtAsOf(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	asOf *time.Time,
	bundle *KlinePatternMTFBundle,
	allowNilAsOfRetry bool,
) {
	if bundle == nil || run == nil {
		return
	}
	key := klinePatternMTFTimeframeKey("daily")
	if ser, ok := bundle.ByFreq[key]; ok && len(ser.Bars) > 0 &&
		len(ser.PatternHits) == len(ser.Bars) && patternHitsSeriesPopulated(ser.PatternHits) {
		return
	}
	probeFreq := klinePatternProbeFrequency("daily")
	m := CapMonthsBack(months)
	bars, hits, ok, unsup := fetchKlinePatternProbeSeriesWithRetry(ctx, run, code, probeFreq, m, asOf)
	if !ok || len(bars) == 0 {
		if allowNilAsOfRetry {
			bars, hits, ok, unsup = fetchKlinePatternProbeSeriesWithRetry(ctx, run, code, probeFreq, m, nil)
		}
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
	HydrateDailyMacroMTFFromBarsForCalendarDay(ctx, run, code, months, dailyBars, bundle, "", nil)
}

// HydrateDailyMacroMTFFromBarsForCalendarDay backfills pattern_hits; skips reuse when calendarDay is set and bars are not aligned to that exact day index.
func HydrateDailyMacroMTFFromBarsForCalendarDay(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	dailyBars []any,
	bundle *KlinePatternMTFBundle,
	calendarDay string,
	sessionAsOf *time.Time,
) {
	if bundle == nil || run == nil || len(dailyBars) == 0 {
		return
	}
	if bundle.ByFreq == nil {
		bundle.ByFreq = map[string]KlinePatternFreqSeries{}
	}
	key := klinePatternMTFTimeframeKey("daily")
	ser := bundle.ByFreq[key]
	reuse := len(ser.Bars) > 0 && len(ser.PatternHits) == len(ser.Bars) &&
		patternHitsSeriesPopulated(ser.PatternHits) && barSeriesSameTimes(ser.Bars, dailyBars)
	if calendarDay != "" || sessionAsOf != nil {
		reuse = false
	}
	if reuse {
		return
	}
	allowNil := sessionAsOf == nil
	hits := ensurePatternHitsOnDailyBarsAt(ctx, run, code, months, dailyBars, ser.PatternHits, sessionAsOf, allowNil)
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