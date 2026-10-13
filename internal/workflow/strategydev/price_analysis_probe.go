package strategydev

import (
	"context"
	"strings"
	"time"
)

// PriceAnalysisProbeParams builds probe_bot_signal_series args for multi-TF price analysis.
// When AsOf is nil, GeeGooSignal returns the latest window from "now" (legacy diagnose card path).
// When AsOf is set, Signal must support as_of/end (GeeGooSignal PR) so bars end at that instant.
type PriceAnalysisProbeParams struct {
	Code   string
	Freq   string
	Months int
	AsOf   *time.Time
}

// PriceAnalysisProbeArgsFrom returns probe args; omits as_of when unset so old Signal stays valid.
func PriceAnalysisProbeArgsFrom(p PriceAnalysisProbeParams) map[string]any {
	args := PriceAnalysisProbeArgs(p.Code, p.Freq, p.Months)
	if p.AsOf != nil && !p.AsOf.IsZero() {
		args["as_of"] = formatPriceAnalysisAsOf(*p.AsOf)
	}
	return args
}

// formatPriceAnalysisAsOf matches GeeGooSignal series probe (Asia/Shanghai wall clock).
func formatPriceAnalysisAsOf(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

// MultiTimeframePriceBarsInput configures a single batched probe per standard timeframe.
type MultiTimeframePriceBarsInput struct {
	Run    ToolRunner
	Code   string
	Months int
	// AsOf ends each probe window at this instant (diagnosis chart end or single-point date).
	AsOf *time.Time
	// ChartBars when set, extends months_back per TF to cover earliest bar + prior-day anchor buffer.
	ChartBars []any
}

// FetchMultiTimeframePriceAnalysisBars probes 5m/60m/daily/weekly once each and caches by frequency code.
func FetchMultiTimeframePriceAnalysisBars(ctx context.Context, in MultiTimeframePriceBarsInput) (barsByFreq map[string][]any, fetchOKByFreq map[string]bool) {
	barsByFreq = map[string][]any{}
	fetchOKByFreq = map[string]bool{}
	if in.Run == nil || strings.TrimSpace(in.Code) == "" {
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
	for _, spec := range multiTimeframeSpecs {
		months := chartMonths
		if len(in.ChartBars) > 0 {
			months = monthsBackForPerSignalPriceFetch(chartMonths, in.ChartBars, spec.Freq, windowEnd)
		}
		bars, ok := fetchPriceAnalysisBarsWithAsOf(ctx, in.Run, in.Code, spec.Freq, months, asOfPtr)
		barsByFreq[spec.Freq] = bars
		fetchOKByFreq[spec.Freq] = ok
	}
	return barsByFreq, fetchOKByFreq
}
