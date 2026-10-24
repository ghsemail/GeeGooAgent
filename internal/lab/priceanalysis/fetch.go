package priceanalysis

import (
	"context"
	"strings"
	"sync"
	"time"
)

const labMultiTFetchParallelism = 3

// BarsFetcher loads OHLCV series via GeeGooSignal lab-api (Agent lab module only).
type BarsFetcher interface {
	FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) (bars []any, ok bool)
}

// MultiTFBarsInput configures one batched multi-timeframe fetch (signal period + coarser only).
type MultiTFBarsInput struct {
	Fetch    BarsFetcher
	Code     string
	Months   int
	GateFreq string
	AsOf     *time.Time
}

// FetchMultiTimeframeBars probes each upward standard TF once.
func FetchMultiTimeframeBars(ctx context.Context, in MultiTFBarsInput) (barsByFreq map[string][]any, fetchOKByFreq map[string]bool, fetchErrByFreq map[string]error) {
	barsByFreq = map[string][]any{}
	fetchOKByFreq = map[string]bool{}
	fetchErrByFreq = map[string]error{}
	if in.Fetch == nil || strings.TrimSpace(in.Code) == "" {
		return barsByFreq, fetchOKByFreq, fetchErrByFreq
	}
	specs := multiTimeframeSpecsForGate(in.GateFreq)
	type tfResult struct {
		freq string
		bars []any
		ok   bool
		err  error
	}
	results := make([]tfResult, len(specs))
	sem := make(chan struct{}, labMultiTFetchParallelism)
	var wg sync.WaitGroup
	for i, spec := range specs {
		i, spec := i, spec
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[i] = tfResult{freq: spec.Freq, ok: false}
				return
			}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				results[i] = tfResult{freq: spec.Freq, ok: false}
				return
			}
			monthsBack, err := MonthsBackForLabFetch(in.Months, spec.Freq)
			if err != nil {
				results[i] = tfResult{freq: spec.Freq, err: err, ok: false}
				return
			}
			bars, ok := in.Fetch.FetchBars(ctx, in.Code, spec.Freq, monthsBack, nil)
			results[i] = tfResult{freq: spec.Freq, bars: bars, ok: ok}
		}()
	}
	wg.Wait()
	for _, r := range results {
		if r.err != nil {
			fetchErrByFreq[r.freq] = r.err
			fetchOKByFreq[r.freq] = false
			continue
		}
		barsByFreq[r.freq] = r.bars
		fetchOKByFreq[r.freq] = r.ok
	}
	return barsByFreq, fetchOKByFreq, fetchErrByFreq
}
