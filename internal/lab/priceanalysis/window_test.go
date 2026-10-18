package priceanalysis

import (
	"fmt"
	"testing"
	"time"
)

func mkOHLCVBarSeries(n int, timeFmt string) []any {
	out := make([]any, n)
	for i := 0; i < n; i++ {
		c := 100.0 + float64(i)*0.05
		ts := fmt.Sprintf(timeFmt, i+1)
		out[i] = map[string]any{
			"time": ts, "close": c, "high": c + 1, "low": c - 1, "volume": 1000.0,
		}
	}
	return out
}

func TestAnalysisBarsLimitDefaults(t *testing.T) {
	cases := map[string]int{
		"5m": 330, "15m": 130, "60m": 130, "daily": 60, "weekly": 26, "1wk": 26,
	}
	for freq, want := range cases {
		if got := analysisBarsLimit(freq); got != want {
			t.Fatalf("%s limit=%d want %d", freq, got, want)
		}
	}
}

func TestTailBarsForPriceAnalysisLongSeries(t *testing.T) {
	long := mkOHLCVBarSeries(500, "2026-01-%02d 10:00:00")
	for _, freq := range []string{"5m", "60m", "daily", "weekly"} {
		tail := tailBarsForPriceAnalysis(long, freq)
		limit := analysisBarsLimit(freq)
		if len(tail) != limit {
			t.Fatalf("%s tail len=%d want %d", freq, len(tail), limit)
		}
	}
}

func TestTailBarsForPriceAnalysisShortSeriesUnchanged(t *testing.T) {
	short := mkOHLCVBarSeries(50, "2026-02-%02d 10:00:00")
	tail := tailBarsForPriceAnalysis(short, "5m")
	if len(tail) != 50 {
		t.Fatalf("short series changed: len=%d", len(tail))
	}
}

func TestTruncateThenTailRespectsAnalysisLimit(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 6, 23, 16, 0, 0, 0, loc)
	long5m := mkOHLCVBarSeries(400, "2026-05-%02d 10:00:00")
	trunc := truncateBarsAtAnchor(long5m, anchor)
	analysisBars := tailBarsForPriceAnalysis(trunc, "5m")
	if len(analysisBars) > analysisBarsLimit("5m") {
		t.Fatalf("tail len=%d want ≤%d", len(analysisBars), analysisBarsLimit("5m"))
	}
	pf := AnalyzePriceFilter(analysisBars, "5m")
	if pf.BarCount != len(analysisBars) {
		t.Fatalf("bar count mismatch pf=%d bars=%d", pf.BarCount, len(analysisBars))
	}
}

func TestTailThenAnalyzeInsufficientStillUnclear(t *testing.T) {
	short := mkOHLCVBarSeries(10, "2026-03-%02d 10:00:00")
	pf := AnalyzePriceFilter(tailBarsForPriceAnalysis(short, "60m"), "60m")
	if pf.Regime != RegimeUnclear {
		t.Fatalf("regime=%s want 不明确", pf.Regime)
	}
	if pf.BarCount != 10 {
		t.Fatalf("bar count=%d", pf.BarCount)
	}
}
