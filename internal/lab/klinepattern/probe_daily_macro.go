package klinepattern

import "context"

// BuildKlinePatternDailyMacroBundle probes only daily bars for macro lab-detail (faster than full MTF).
func BuildKlinePatternDailyMacroBundle(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	chartBars []any,
) *KlinePatternMTFBundle {
	if run == nil {
		return nil
	}
	spec := klinePatternMTFSpec{Label: "日线", Freq: "daily"}
	bundle := &KlinePatternMTFBundle{
		SignalFreq: "daily",
		ByFreq:     map[string]KlinePatternFreqSeries{},
	}
	windowEnd := chartWindowEnd(chartBars)
	chartMonths := CapMonthsBack(months)
	m := chartMonths
	if len(chartBars) > 0 && host.MonthsBackForPerSignalFetch != nil {
		m = host.MonthsBackForPerSignalFetch(chartMonths, chartBars, spec.Freq, windowEnd)
	}
	probeFreq := klinePatternProbeFrequency(spec.Freq)
	bars, hits, ok, unsup := fetchKlinePatternProbeSeriesWithRetry(ctx, run, code, probeFreq, m, nil)
	bundle.ByFreq[klinePatternMTFTimeframeKey(spec.Freq)] = KlinePatternFreqSeries{
		Label: spec.Label, Freq: spec.Freq, Bars: bars, PatternHits: hits,
		FetchOK: ok, Unsupported: unsup,
	}
	return bundle
}
