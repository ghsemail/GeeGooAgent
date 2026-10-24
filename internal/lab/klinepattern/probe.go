package klinepattern

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// KlinePatternFreqSeries is OHLCV + pattern_hits for one timeframe probe.
type KlinePatternFreqSeries struct {
	Label        string
	Freq         string
	Bars         []any
	PatternHits  []any
	FetchOK      bool
	Unsupported  bool // probe/Data rejected frequency (e.g. 240m)
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
) (bars, patternHits []any, ok, unsupported bool) {
	if run == nil || host.ProbeArgsExtended == nil {
		return nil, nil, false, false
	}
	res := run(ctx, "probe_bot_signal_series", host.ProbeArgsExtended(code, freq, months, asOf))
	if res.Status != tools.StatusOK {
		return nil, nil, false, toolsResultUnsupportedFrequency(res)
	}
	bars, _ = res.Data["bars"].([]any)
	if ph, ok := res.Data["pattern_hits"].([]any); ok {
		patternHits = ph
	}
	return bars, patternHits, len(bars) > 0, false
}

func toolsResultUnsupportedFrequency(res tools.Result) bool {
	msg := strings.ToLower(strings.TrimSpace(res.Summary))
	return strings.Contains(msg, "unsupported frequency") || strings.Contains(msg, "unsupported")
}

// BuildKlinePatternMTFBundle probes signal + higher TFs once (display-only background).
func BuildKlinePatternMTFBundle(
	ctx context.Context,
	run ToolRunner,
	code, signalFreq string,
	months int,
	chartBars []any,
) *KlinePatternMTFBundle {
	if run == nil {
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
	chartMonths := CapMonthsBack(months)
	for _, spec := range multiTimeframeSpecsForKlinePatternDisplay(signalFreq) {
		m := chartMonths
		if len(chartBars) > 0 && host.MonthsBackForPerSignalFetch != nil {
			m = host.MonthsBackForPerSignalFetch(chartMonths, chartBars, spec.Freq, windowEnd)
		}
		probeFreq := klinePatternProbeFrequency(spec.Freq)
		// Full probe window; per-episode alignment truncates locally in BuildKlinePatternMTFContext.
		bars, hits, ok, unsup := fetchKlinePatternProbeSeriesWithRetry(ctx, run, code, probeFreq, m, nil)
		bundle.ByFreq[klinePatternMTFTimeframeKey(spec.Freq)] = KlinePatternFreqSeries{
			Label: spec.Label, Freq: spec.Freq, Bars: bars, PatternHits: hits,
			FetchOK: ok, Unsupported: unsup,
		}
	}
	return bundle
}
