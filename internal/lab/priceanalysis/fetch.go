package priceanalysis

import (
	"context"
	"strings"
	"time"
)

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
	for _, spec := range multiTimeframeSpecsForGate(in.GateFreq) {
		monthsBack, err := MonthsBackForLabFetch(in.Months, spec.Freq)
		if err != nil {
			fetchErrByFreq[spec.Freq] = err
			fetchOKByFreq[spec.Freq] = false
			continue
		}
		// Full lab fetch window to latest; Run truncates at req.AsOf locally per episode.
		bars, ok := in.Fetch.FetchBars(ctx, in.Code, spec.Freq, monthsBack, nil)
		barsByFreq[spec.Freq] = bars
		fetchOKByFreq[spec.Freq] = ok
	}
	return barsByFreq, fetchOKByFreq, fetchErrByFreq
}
