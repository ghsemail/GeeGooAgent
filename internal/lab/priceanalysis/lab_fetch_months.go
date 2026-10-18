package priceanalysis

import (
	"fmt"
	"math"
)

// LabKlineMaxBarsPerJob matches GeeGooSignal lab-api single-task bar cap (no 800 wrapper limit).
const LabKlineMaxBarsPerJob = 5000

// MonthsBackForLabFetch computes calendar months_back for POST /v1/lab/kline/jobs per timeframe.
//
//	months_back(tf) = 信号锚点跨度(months) + 该周期分析窗所需月数 + 预热余量
//
// 分析窗（锚点截断后 tailBarsForPriceAnalysis）仍为定稿根数；取数窗仅保证最早锚点处也能凑够 tail+minBars。
func MonthsBackForLabFetch(signalMonths int, freq string) (int, error) {
	anchor := CapMonthsBack(signalMonths)
	analysisM := analysisWindowCalendarMonths(freq)
	warmM := warmupCalendarMonths(freq)
	total := anchor + analysisM + warmM
	if total < 1 {
		total = 1
	}
	minBars := minFetchBarsRequired(freq)
	if est := EstimateFetchBars(freq, total); est > LabKlineMaxBarsPerJob {
		capped := maxMonthsBackWithinBarCap(freq, LabKlineMaxBarsPerJob)
		if capped < 1 || EstimateFetchBars(freq, capped) < minBars {
			return 0, fmt.Errorf(
				"lab kline %s: need about %d bars for %dmo signal span but lab-api cap is %d bars (~%dmo); shorten signal months or split timeframe fetches",
				NormalizeFrequency(freq), est, anchor, LabKlineMaxBarsPerJob, capped,
			)
		}
		total = capped
	}
	if EstimateFetchBars(freq, total) < minBars {
		return 0, fmt.Errorf(
			"lab kline %s: %d calendar months yields ~%d bars, need at least %d for analysis",
			NormalizeFrequency(freq), total, EstimateFetchBars(freq, total), minBars,
		)
	}
	return total, nil
}

func minFetchBarsRequired(freq string) int {
	want := analysisBarsLimit(freq)
	if mb := minBarsForPriceFilter(freq); mb > want {
		want = mb
	}
	return want
}

// analysisWindowCalendarMonths is calendar months covered by the analysis tail window (定稿根数).
func analysisWindowCalendarMonths(freq string) int {
	switch NormalizeFrequency(freq) {
	case "5m":
		return 1 // ~330 bars ≈ 1 week; fetch calendar uses 1mo buffer
	case "60m", "15m":
		return 1 // ~130 bars ≈ 1 month
	case "daily":
		return 3 // ~60 bars ≈ 2–3 months
	case "weekly", "1wk":
		return 6 // ~26 bars ≈ half year
	default:
		return 2
	}
}

func warmupCalendarMonths(freq string) int {
	switch NormalizeFrequency(freq) {
	case "weekly", "1wk":
		return 1
	default:
		return 1
	}
}

// EstimateFetchBars approximates bar count returned for months_back (conservative, for cap checks).
func EstimateFetchBars(freq string, monthsBack int) int {
	if monthsBack <= 0 {
		return 0
	}
	switch NormalizeFrequency(freq) {
	case "5m":
		return monthsBack * 22 * 48
	case "60m", "15m":
		return monthsBack * 22 * 6
	case "daily":
		return monthsBack * 22
	case "weekly", "1wk":
		return int(math.Ceil(float64(monthsBack) * 4.33))
	default:
		return monthsBack * 22 * 6
	}
}

func maxMonthsBackWithinBarCap(freq string, cap int) int {
	if cap <= 0 {
		return 0
	}
	for m := 1; m <= 120; m++ {
		if EstimateFetchBars(freq, m+1) > cap {
			return m
		}
	}
	return 120
}
