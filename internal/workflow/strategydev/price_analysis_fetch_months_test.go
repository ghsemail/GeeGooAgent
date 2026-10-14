package strategydev

import (
	"testing"
	"time"
)

func TestMonthsBackForPerSignalPriceFetchExtendsShortChartWindow(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// Chart spans June–October but UI months_back is only 3.
	chart := []any{
		map[string]any{"time": "2026-06-24 11:00:00", "close": 428.8, "high": 430, "low": 427, "volume": 1},
		map[string]any{"time": "2026-10-10 11:00:00", "close": 440.0, "high": 441, "low": 439, "volume": 1},
	}
	end := time.Date(2026, 10, 10, 11, 0, 0, 0, loc)
	got := monthsBackForPerSignalPriceFetch(3, chart, "daily", end)
	if got <= 3 {
		t.Fatalf("daily months_back=%d want > 3 to cover June anchor", got)
	}
	if got > 6 {
		t.Fatalf("daily months_back=%d want capped at 6", got)
	}
}

func TestMonthsBackForPerSignalPriceFetchWeeklyNeedsLongerSpan(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	chart := []any{
		map[string]any{"time": "2026-06-01 11:00:00", "close": 400.0, "high": 401, "low": 399, "volume": 1},
		map[string]any{"time": "2026-10-01 11:00:00", "close": 420.0, "high": 421, "low": 419, "volume": 1},
	}
	end := time.Date(2026, 10, 1, 11, 0, 0, 0, loc)
	got := monthsBackForPerSignalPriceFetch(3, chart, "weekly", end)
	if got < 5 {
		t.Fatalf("weekly months_back=%d want enough for min weekly bars", got)
	}
}

func TestMonthsBackForPerSignalPriceFetchThreeMonthChartJulyAnchorWeekly(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// 60m diagnose with months_back=3 but bars span Jul–Oct; early signal anchor ~Jul 24 needs
	// ~26 weekly bars before truncate — must exceed CapMonthsBack(6) on the probe.
	chart := []any{
		map[string]any{"time": "2026-07-01 10:00:00", "close": 210.0, "high": 211, "low": 209, "volume": 1},
		map[string]any{"time": "2026-10-10 11:00:00", "close": 230.0, "high": 231, "low": 229, "volume": 1},
	}
	end := time.Date(2026, 10, 10, 11, 0, 0, 0, loc)
	got := monthsBackForPerSignalPriceFetch(3, chart, "weekly", end)
	if got <= 6 {
		t.Fatalf("weekly months_back=%d want >6 for Jul anchor + 26-bar analysis tail", got)
	}
	args := PriceAnalysisProbeArgs("AAPL", "weekly", got)
	if args["months_back"].(int) != got {
		t.Fatalf("probe months_back=%v want uncapped weekly fetch %d", args["months_back"], got)
	}
}
