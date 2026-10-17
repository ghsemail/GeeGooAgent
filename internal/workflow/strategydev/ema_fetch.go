package strategydev

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const (
	emaDailyFetchMonthsFloor  = 18 // ~250 trading days
	emaWeeklyFetchMonthsFloor = 36 // ~150 weeks
	ema60mFetchBarLimitDefault = 900
	ema60mFetchBarLimitMin     = 800
	ema60mFetchBarLimitMax     = 1000
)

// capMonthsBackForEMAFetch enforces minimum calendar spans for EMA display probes (no 12mo PA cap).
func capMonthsBackForEMAFetch(freq string, months int) int {
	if months <= 0 {
		months = 3
	}
	switch NormalizeFrequency(freq) {
	case "daily":
		if months < emaDailyFetchMonthsFloor {
			months = emaDailyFetchMonthsFloor
		}
	case "weekly", "1wk":
		if months < emaWeeklyFetchMonthsFloor {
			months = emaWeeklyFetchMonthsFloor
		}
	case "60m":
		if months < perSignalPriceFetchMonthsCap {
			months = perSignalPriceFetchMonthsCap
		}
	}
	return months
}

func emaMinBarsForFetch(freq string) int {
	switch NormalizeFrequency(freq) {
	case "weekly", "1wk":
		return 150
	case "daily":
		return 250
	case "60m":
		return ema60mFetchBarLimitDefault
	default:
		return emaMinBarsForSlow
	}
}

func emaMinBarCalendarLookback(freq string) time.Duration {
	minBars := emaMinBarsForFetch(freq)
	switch NormalizeFrequency(freq) {
	case "5m":
		days := (minBars / 48) + 3
		return time.Duration(days) * 24 * time.Hour
	case "60m":
		days := (minBars / 4) + 10
		return time.Duration(days) * 24 * time.Hour
	case "daily":
		days := minBars + 10
		return time.Duration(days) * 24 * time.Hour
	case "weekly", "1wk":
		weeks := minBars + 4
		return time.Duration(weeks*7) * 24 * time.Hour
	default:
		return 35 * 24 * time.Hour
	}
}

// monthsBackForEMAFetch extends chart months_back for EMA stacks (separate from PA 12mo cap).
func monthsBackForEMAFetch(chartMonths int, chartBars []any, freq string, windowEnd time.Time) int {
	base := CapMonthsBack(chartMonths)
	earliest, latest := barSeriesTimeExtent(chartBars)
	if earliest.IsZero() {
		return capMonthsBackForEMAFetch(freq, base)
	}
	if !latest.IsZero() && (windowEnd.IsZero() || latest.After(windowEnd)) {
		windowEnd = latest
	}
	if windowEnd.IsZero() {
		windowEnd = latest
	}
	if windowEnd.IsZero() {
		return capMonthsBackForEMAFetch(freq, base)
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	needStart := earliest.In(loc).AddDate(0, 0, -priorTradingDayCalendarBufferDays)
	needStart = needStart.Add(-emaMinBarCalendarLookback(freq))

	spanMonths := ceilCalendarMonthsBetween(needStart, windowEnd.In(loc))
	if spanMonths > base {
		base = spanMonths
	}
	return capMonthsBackForEMAFetch(freq, base)
}

func ema60mFetchBarLimit(chartBars []any, windowEnd time.Time) int {
	limit := ema60mFetchBarLimitDefault
	earliest, latest := barSeriesTimeExtent(chartBars)
	if earliest.IsZero() {
		return clampInt(limit, ema60mFetchBarLimitMin, ema60mFetchBarLimitMax)
	}
	if !latest.IsZero() && (windowEnd.IsZero() || latest.After(windowEnd)) {
		windowEnd = latest
	}
	if windowEnd.IsZero() {
		return clampInt(limit, ema60mFetchBarLimitMin, ema60mFetchBarLimitMax)
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	needStart := earliest.In(loc).AddDate(0, 0, -priorTradingDayCalendarBufferDays)
	needStart = needStart.Add(-emaMinBarCalendarLookback("60m"))
	if windowEnd.After(needStart) {
		days := int(windowEnd.Sub(needStart).Hours()/24) + 1
		needBars := days*4 + 20
		if needBars > limit {
			limit = needBars
		}
	}
	return clampInt(limit, ema60mFetchBarLimitMin, ema60mFetchBarLimitMax)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// MultiTimeframeEMABarsInput configures EMA-only OHLCV probes (display-only; not price analysis).
type MultiTimeframeEMABarsInput struct {
	Run       ToolRunner
	Code      string
	Months    int
	AsOf      *time.Time
	ChartBars []any
}

// FetchMultiTimeframeEMABars probes weekly / daily / 60m with EMA-specific fetch margins.
func FetchMultiTimeframeEMABars(ctx context.Context, in MultiTimeframeEMABarsInput) (barsByFreq map[string][]any, fetchOKByFreq map[string]bool) {
	barsByFreq = map[string][]any{}
	fetchOKByFreq = map[string]bool{}
	if !PriceAnalysisFeatureEnabled() {
		return barsByFreq, fetchOKByFreq
	}
	ctx = ContextWithPriceProbeCache(ctx)
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
	for _, spec := range emaStackTimeframes {
		freq := NormalizeFrequency(spec.freq)
		if freq == "60m" {
			limit := ema60mFetchBarLimit(in.ChartBars, windowEnd)
			months := monthsBackForEMAFetch(chartMonths, in.ChartBars, freq, windowEnd)
			bars, ok := fetchEMABarsWithAsOf(ctx, in.Run, in.Code, freq, months, limit, asOfPtr)
			barsByFreq[freq] = bars
			fetchOKByFreq[freq] = ok
			continue
		}
		months := monthsBackForEMAFetch(chartMonths, in.ChartBars, freq, windowEnd)
		if len(in.ChartBars) == 0 {
			months = capMonthsBackForEMAFetch(freq, chartMonths)
		}
		bars, ok := fetchEMABarsWithAsOf(ctx, in.Run, in.Code, freq, months, 0, asOfPtr)
		barsByFreq[freq] = bars
		fetchOKByFreq[freq] = ok
	}
	return barsByFreq, fetchOKByFreq
}

func emaProbeArgsFrom(code, freq string, months, limit int, asOf *time.Time) map[string]any {
	months = capMonthsBackForEMAFetch(freq, months)
	args := map[string]any{
		"code":        strings.TrimSpace(code),
		"frequency":   NormalizeFrequency(freq),
		"months_back": months,
		"buy_signal":  slots.NormalizeSignalChain(nosignalProbe),
		"sell_signal": slots.NormalizeSignalChain(nosignalProbe),
	}
	if limit > 0 {
		args["limit"] = limit
	}
	if asOf != nil && !asOf.IsZero() {
		args["as_of"] = formatPriceAnalysisAsOf(*asOf)
	}
	return args
}

// isEMAStackProbeArgs detects EMA display probes (longer fetch than price analysis).
func isEMAStackProbeArgs(args map[string]any) bool {
	if args == nil {
		return false
	}
	if lim, ok := args["limit"].(int); ok && lim >= ema60mFetchBarLimitMin {
		return true
	}
	if lim, ok := args["limit"].(float64); ok && int(lim) >= ema60mFetchBarLimitMin {
		return true
	}
	months, _ := args["months_back"].(int)
	if months == 0 {
		if mf, ok := args["months_back"].(float64); ok {
			months = int(mf)
		}
	}
	freq, _ := args["frequency"].(string)
	switch NormalizeFrequency(freq) {
	case "daily":
		return months >= emaDailyFetchMonthsFloor
	case "weekly", "1wk":
		return months >= emaWeeklyFetchMonthsFloor
	default:
		return false
	}
}

func fetchEMABarsWithAsOf(ctx context.Context, run ToolRunner, code, freq string, months, limit int, asOf *time.Time) ([]any, bool) {
	if run == nil {
		return nil, false
	}
	key := probeCacheKey(code, freq, months, limit, asOf)
	if bars, ok, hit := lookupPriceProbeCache(ctx, key); hit {
		return bars, ok
	}
	var lastSummary string
	for attempt := 1; attempt <= priceAnalysisProbeMaxAttempts; attempt++ {
		res := run(ctx, "probe_bot_signal_series", emaProbeArgsFrom(code, freq, months, limit, asOf))
		if res.Status == tools.StatusOK {
			bars, _ := res.Data["bars"].([]any)
			storePriceProbeCache(ctx, key, bars, true)
			return bars, true
		}
		lastSummary = res.Summary
		if attempt < priceAnalysisProbeMaxAttempts && ctx.Err() == nil {
			delay := priceAnalysisProbeRetryBase * time.Duration(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				storePriceProbeCache(ctx, key, nil, false)
				return nil, false
			case <-timer.C:
			}
		}
	}
	_ = lastSummary
	storePriceProbeCache(ctx, key, nil, false)
	return nil, false
}
