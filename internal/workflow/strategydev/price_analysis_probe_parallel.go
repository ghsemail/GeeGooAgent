package strategydev

import (
	"context"
	"sync"
	"time"
)

const priceAnalysisProbeParallelism = 4

// FetchMultiTimeframePriceAnalysisBarsParallel probes standard TFs concurrently (when feature enabled).
func FetchMultiTimeframePriceAnalysisBarsParallel(ctx context.Context, in MultiTimeframePriceBarsInput) (barsByFreq map[string][]any, fetchOKByFreq map[string]bool) {
	if !PriceAnalysisFeatureEnabled() {
		return map[string][]any{}, map[string]bool{}
	}
	barsByFreq = map[string][]any{}
	fetchOKByFreq = map[string]bool{}
	ctx = ContextWithPriceProbeCache(ctx)
	if in.Run == nil || in.Code == "" {
		return barsByFreq, fetchOKByFreq
	}
	chartMonths := CapMonthsBack(in.Months)
	var windowEnd time.Time
	if in.AsOf != nil {
		windowEnd = *in.AsOf
	}
	if len(in.ChartBars) > 0 {
		_, latest := barSeriesTimeExtent(in.ChartBars)
		if !latest.IsZero() && (windowEnd.IsZero() || latest.After(windowEnd)) {
			windowEnd = latest
		}
	}
	var asOfPtr *time.Time
	if !windowEnd.IsZero() {
		end := windowEnd
		asOfPtr = &end
	}
	specs := multiTimeframeSpecsForGate(in.GateFreq)
	type result struct {
		freq string
		bars []any
		ok   bool
	}
	results := make([]result, len(specs))
	sem := make(chan struct{}, priceAnalysisProbeParallelism)
	var wg sync.WaitGroup
	for i, spec := range specs {
		i, spec := i, spec
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			months := chartMonths
			if len(in.ChartBars) > 0 {
				months = monthsBackForPerSignalPriceFetch(chartMonths, in.ChartBars, spec.Freq, windowEnd)
			}
			bars, ok := fetchPriceAnalysisBarsWithAsOf(ctx, in.Run, in.Code, spec.Freq, months, asOfPtr)
			results[i] = result{freq: spec.Freq, bars: bars, ok: ok}
		}()
	}
	wg.Wait()
	for _, r := range results {
		barsByFreq[r.freq] = r.bars
		fetchOKByFreq[r.freq] = r.ok
	}
	return barsByFreq, fetchOKByFreq
}
