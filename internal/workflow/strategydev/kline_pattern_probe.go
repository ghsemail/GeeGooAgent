package strategydev

import (
	"context"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// KlinePatternFreqSeries is OHLCV + pattern_hits for one timeframe probe.
type KlinePatternFreqSeries struct {
	Label       string
	Freq        string
	Bars        []any
	PatternHits []any
	FetchOK     bool
}

// KlinePatternMTFBundle holds upward timeframes for display-only MTF pattern context.
type KlinePatternMTFBundle struct {
	SignalFreq string
	ByFreq     map[string]KlinePatternFreqSeries
}

func fetchKlinePatternProbeSeries(
	ctx context.Context,
	run ToolRunner,
	code, freq string,
	months int,
	asOf *time.Time,
) (bars, patternHits []any, ok bool) {
	if !KlinePatternDisplayEnabled() || run == nil {
		return nil, nil, false
	}
	res := run(ctx, "probe_bot_signal_series", PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{
		Code: code, Freq: freq, Months: months, AsOf: asOf, ExtendedFetch: true,
	}))
	if res.Status != tools.StatusOK {
		return nil, nil, false
	}
	bars, _ = res.Data["bars"].([]any)
	if ph, ok := res.Data["pattern_hits"].([]any); ok {
		patternHits = ph
	}
	return bars, patternHits, len(bars) > 0
}

// BuildKlinePatternMTFBundle probes signal + higher TFs once (display-only background).
func BuildKlinePatternMTFBundle(
	ctx context.Context,
	run ToolRunner,
	code, signalFreq string,
	months int,
	chartBars []any,
) *KlinePatternMTFBundle {
	if !KlinePatternDisplayEnabled() || run == nil {
		return nil
	}
	signalFreq = NormalizeFrequency(signalFreq)
	bundle := &KlinePatternMTFBundle{
		SignalFreq: signalFreq,
		ByFreq:     map[string]KlinePatternFreqSeries{},
	}
	var windowEnd time.Time
	if len(chartBars) > 0 {
		_, latest := barSeriesTimeExtent(chartBars)
		windowEnd = latest
	}
	var asOf *time.Time
	if !windowEnd.IsZero() {
		end := windowEnd
		asOf = &end
	}
	chartMonths := CapMonthsBack(months)
	for _, spec := range multiTimeframeSpecsForGate(signalFreq) {
		m := chartMonths
		if len(chartBars) > 0 {
			m = monthsBackForPerSignalPriceFetch(chartMonths, chartBars, spec.Freq, windowEnd)
		}
		bars, hits, ok := fetchKlinePatternProbeSeries(ctx, run, code, spec.Freq, m, asOf)
		bundle.ByFreq[spec.Freq] = KlinePatternFreqSeries{
			Label: spec.Label, Freq: spec.Freq, Bars: bars, PatternHits: hits, FetchOK: ok,
		}
	}
	return bundle
}
