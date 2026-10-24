package klinepattern

import (
	"context"
	"sync"
	"time"
)

const klinePatternProbeParallelism = 3

// BuildKlinePatternMTFBundleParallel is like BuildKlinePatternMTFBundle but probes TFs concurrently (capped).
func BuildKlinePatternMTFBundleParallel(
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
	specs := multiTimeframeSpecsForKlinePatternDisplay(signalFreq)
	if len(specs) == 0 {
		return nil
	}
	bundle := &KlinePatternMTFBundle{
		SignalFreq: signalFreq,
		ByFreq:     map[string]KlinePatternFreqSeries{},
	}
	var windowEnd = chartWindowEnd(chartBars)
	chartMonths := CapMonthsBack(months)

	type job struct {
		spec klinePatternMTFSpec
	}
	jobs := make([]job, len(specs))
	for i, spec := range specs {
		jobs[i] = job{spec: spec}
	}
	sem := make(chan struct{}, klinePatternProbeParallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			m := chartMonths
			if len(chartBars) > 0 && host.MonthsBackForPerSignalFetch != nil {
				m = host.MonthsBackForPerSignalFetch(chartMonths, chartBars, j.spec.Freq, windowEnd)
			}
			probeFreq := klinePatternProbeFrequency(j.spec.Freq)
			bars, hits, ok, unsup := fetchKlinePatternProbeSeries(ctx, run, code, probeFreq, m, nil)
			ser := KlinePatternFreqSeries{
				Label: j.spec.Label, Freq: j.spec.Freq, Bars: bars, PatternHits: hits,
				FetchOK: ok, Unsupported: unsup,
			}
			mu.Lock()
			bundle.ByFreq[klinePatternMTFTimeframeKey(j.spec.Freq)] = ser
			mu.Unlock()
		}()
	}
	wg.Wait()
	return bundle
}

func chartWindowEnd(chartBars []any) (windowEnd time.Time) {
	if len(chartBars) > 0 {
		_, windowEnd = barSeriesTimeExtent(chartBars)
	}
	return windowEnd
}

