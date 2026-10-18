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
func FetchMultiTimeframeBars(ctx context.Context, in MultiTFBarsInput) (barsByFreq map[string][]any, fetchOKByFreq map[string]bool) {
	barsByFreq = map[string][]any{}
	fetchOKByFreq = map[string]bool{}
	if in.Fetch == nil || strings.TrimSpace(in.Code) == "" {
		return barsByFreq, fetchOKByFreq
	}
	months := CapMonthsBack(in.Months)
	var asOfPtr *time.Time
	if in.AsOf != nil && !in.AsOf.IsZero() {
		end := *in.AsOf
		asOfPtr = &end
	}
	for _, spec := range multiTimeframeSpecsForGate(in.GateFreq) {
		monthsBack := capMonthsBackForPriceAnalysisFetch(spec.Freq, months)
		if asOfPtr != nil {
			monthsBack = capMonthsBackForPerSignalPriceFetch(spec.Freq, monthsBack)
		}
		bars, ok := in.Fetch.FetchBars(ctx, in.Code, spec.Freq, monthsBack, asOfPtr)
		barsByFreq[spec.Freq] = bars
		fetchOKByFreq[spec.Freq] = ok
	}
	return barsByFreq, fetchOKByFreq
}
