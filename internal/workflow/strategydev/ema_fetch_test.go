package strategydev

import (
	"testing"
	"time"
)

func TestCapMonthsBackForEMAFetchFloorsPerTF(t *testing.T) {
	if got := capMonthsBackForEMAFetch("daily", 3); got < emaDailyFetchMonthsFloor {
		t.Fatalf("daily months=%d want floor >= %d", got, emaDailyFetchMonthsFloor)
	}
	if got := capMonthsBackForEMAFetch("weekly", 6); got < emaWeeklyFetchMonthsFloor {
		t.Fatalf("weekly months=%d want floor >= %d", got, emaWeeklyFetchMonthsFloor)
	}
	if got := capMonthsBackForEMAFetch("60m", 3); got != perSignalPriceFetchMonthsCap {
		t.Fatalf("60m months=%d want %d", got, perSignalPriceFetchMonthsCap)
	}
}

func TestCapMonthsBackForPerSignalPriceFetchStillTwelveMonthCap(t *testing.T) {
	got := capMonthsBackForPerSignalPriceFetch("daily", 24)
	if got != perSignalPriceFetchMonthsCap {
		t.Fatalf("PA daily months=%d want cap %d unchanged", got, perSignalPriceFetchMonthsCap)
	}
	gotWeekly := monthsBackForPerSignalPriceFetch(3, nil, "weekly", time.Time{})
	if gotWeekly > perSignalPriceFetchMonthsCap {
		t.Fatalf("PA weekly months=%d must stay <= %d", gotWeekly, perSignalPriceFetchMonthsCap)
	}
}

func TestMonthsBackForEMAFetchExceedsPACapForEarlyChart(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	chart := []any{
		map[string]any{"time": "2026-06-01 11:00:00", "close": 400.0, "high": 401, "low": 399, "volume": 1},
		map[string]any{"time": "2026-10-10 11:00:00", "close": 440.0, "high": 441, "low": 439, "volume": 1},
	}
	end := time.Date(2026, 10, 10, 11, 0, 0, 0, loc)
	paWeekly := monthsBackForPerSignalPriceFetch(3, chart, "weekly", end)
	emaWeekly := monthsBackForEMAFetch(3, chart, "weekly", end)
	if emaWeekly <= paWeekly || emaWeekly < emaWeeklyFetchMonthsFloor {
		t.Fatalf("ema weekly months=%d want > PA %d and >= floor %d", emaWeekly, paWeekly, emaWeeklyFetchMonthsFloor)
	}
	emaDaily := monthsBackForEMAFetch(3, chart, "daily", end)
	if emaDaily <= perSignalPriceFetchMonthsCap {
		t.Fatalf("ema daily months=%d want > PA cap %d", emaDaily, perSignalPriceFetchMonthsCap)
	}
}

func TestEma60mFetchBarLimitInRange(t *testing.T) {
	limit := ema60mFetchBarLimit(nil, time.Time{})
	if limit < ema60mFetchBarLimitMin || limit > ema60mFetchBarLimitMax {
		t.Fatalf("default limit=%d want [%d,%d]", limit, ema60mFetchBarLimitMin, ema60mFetchBarLimitMax)
	}
}

func TestEMAProbeArgs60mIncludesLimit(t *testing.T) {
	args := emaProbeArgsFrom("AAPL", "60m", 12, 900, nil)
	if args["limit"].(int) != 900 {
		t.Fatalf("limit=%v", args["limit"])
	}
	if args["months_back"].(int) != perSignalPriceFetchMonthsCap {
		t.Fatalf("months_back=%v", args["months_back"])
	}
}

func TestEMAProbeArgsDailyOmitsLimit(t *testing.T) {
	args := emaProbeArgsFrom("AAPL", "daily", 3, 0, nil)
	if _, ok := args["limit"]; ok {
		t.Fatalf("daily must not set limit: %v", args)
	}
	if args["months_back"].(int) != emaDailyFetchMonthsFloor {
		t.Fatalf("months_back=%v want floor %d", args["months_back"], emaDailyFetchMonthsFloor)
	}
}

func TestComputeEMAStacksUsesLongWeeklyHistoryNotPAWindow(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 6, 26, 16, 0, 0, 0, loc)
	makeTrend := func(n int) []any {
		out := make([]any, n)
		for i := 0; i < n; i++ {
			c := 100.0 + float64(i)*0.8
			out[i] = map[string]any{
				"time": time.Date(2024, 1, 1, 0, 0, 0, 0, loc).AddDate(0, 0, i*7).Format("2006-01-02"),
				"close": c, "high": c + 1, "low": c - 1, "volume": 1.0,
			}
		}
		return out
	}
	shortPA := makeTrend(45)
	longEMA := makeTrend(130)
	stacks := ComputeEMAStacksAtAnchor(
		map[string][]any{"weekly": longEMA, "daily": longEMA, "60m": longEMA},
		map[string]bool{"weekly": true, "daily": true, "60m": true},
		nil, "60m", anchor,
	)
	if stacks.Weekly.Label == emaStackLabelUnknown {
		t.Fatalf("weekly should classify with long EMA fetch bars, not PA-short %d bars", len(shortPA))
	}
	stacksShort := ComputeEMAStacksAtAnchor(
		map[string][]any{"weekly": shortPA},
		map[string]bool{"weekly": true},
		nil, "60m", anchor,
	)
	if stacksShort.Weekly.Label != emaStackLabelUnknown {
		t.Fatalf("weekly with %d bars should stay unknown", len(shortPA))
	}
}
