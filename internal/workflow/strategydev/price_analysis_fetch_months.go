package strategydev

import "time"

// Calendar days before the earliest chart bar to resolve prior trading day daily
// anchors across long holidays (A-share).
const priorTradingDayCalendarBufferDays = 14

// monthsBackForPerSignalPriceFetch extends chart months_back so multi-TF probes cover
// the full diagnose chart span plus lookback for prior-day anchors and minBars structure.
func monthsBackForPerSignalPriceFetch(chartMonths int, chartBars []any, freq string, windowEnd time.Time) int {
	base := CapMonthsBack(chartMonths)
	earliest, latest := barSeriesTimeExtent(chartBars)
	if earliest.IsZero() {
		return base
	}
	if !latest.IsZero() && (windowEnd.IsZero() || latest.After(windowEnd)) {
		windowEnd = latest
	}
	if windowEnd.IsZero() {
		windowEnd = latest
	}
	if windowEnd.IsZero() {
		return base
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.UTC
	}
	needStart := earliest.In(loc).AddDate(0, 0, -priorTradingDayCalendarBufferDays)
	needStart = needStart.Add(-minBarCalendarLookback(freq))

	spanMonths := ceilCalendarMonthsBetween(needStart, windowEnd.In(loc))
	if spanMonths > base {
		base = spanMonths
	}
	return capMonthsBackForPriceAnalysisFetch(freq, base)
}

func minBarCalendarLookback(freq string) time.Duration {
	// Conservative calendar span so truncateBarsAtAnchor still leaves enough bars for tail analysis.
	minBars := minBarsForPriceFilter(freq)
	switch NormalizeFrequency(freq) {
	case "weekly", "1wk":
		if want := analysisBarsLimit(freq); want > minBars {
			minBars = want
		}
	}
	switch NormalizeFrequency(freq) {
	case "5m":
		// ~48 A-share 5m bars / session
		days := (minBars / 48) + 3
		return time.Duration(days) * 24 * time.Hour
	case "60m":
		days := (minBars / 4) + 5
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

func barSeriesTimeExtent(bars []any) (earliest, latest time.Time) {
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() {
			continue
		}
		if earliest.IsZero() || ts.Before(earliest) {
			earliest = ts
		}
		if latest.IsZero() || ts.After(latest) {
			latest = ts
		}
	}
	return earliest, latest
}

func ceilCalendarMonthsBetween(start, end time.Time) int {
	if end.Before(start) {
		start, end = end, start
	}
	if !end.After(start) {
		return 1
	}
	months := 0
	cursor := start
	for cursor.Before(end) {
		next := cursor.AddDate(0, 1, 0)
		months++
		if !next.Before(end) {
			break
		}
		cursor = next
	}
	if months < 1 {
		months = 1
	}
	return months
}
