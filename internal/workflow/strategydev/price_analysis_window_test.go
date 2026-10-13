package strategydev

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

func TestAnchoredAnalysisUsesTailAfterTruncate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 6, 23, 16, 0, 0, 0, loc)
	long5m := mkOHLCVBarSeries(400, "2026-05-%02d 10:00:00")
	barsByFreq := map[string][]any{
		"5m": long5m, "60m": long5m, "daily": mkOHLCVBarSeries(80, "2026-04-%02d"), "weekly": mkOHLCVBarSeries(25, "2026-01-%02d"),
	}
	fetchOK := map[string]bool{"5m": true, "60m": true, "daily": true, "weekly": true}
	pa := buildAnchoredPriceAnalysis(barsByFreq, fetchOK, nil, "5m", anchor, 420, 428, KeyLevelBands{})
	var row5m PriceAnalysisRow
	for _, r := range pa.Rows {
		if r.Timeframe == "5m" {
			row5m = r
			break
		}
	}
	trunc := truncateBarsAtAnchor(long5m, anchor)
	analysisBars := tailBarsForPriceAnalysis(trunc, "5m")
	pf := AnalyzePriceFilter(analysisBars, "5m")
	if pf.BarCount > analysisBarsLimit("5m") {
		t.Fatalf("gate bar count=%d want ≤%d", pf.BarCount, analysisBarsLimit("5m"))
	}
	if pf.BarCount != len(analysisBars) {
		t.Fatalf("bar count mismatch pf=%d bars=%d", pf.BarCount, len(analysisBars))
	}
	if row5m.Structure == structureInsufficientBars && len(analysisBars) >= minBarsForPriceFilter("5m") {
		t.Fatalf("unexpected insufficient with %d analysis bars", len(analysisBars))
	}
}

func TestAnchoredGateAnalyzeUsesAtMostAnalysisLimit(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 6, 23, 16, 0, 0, 0, loc)
	long := mkOHLCVBarSeries(350, "2026-05-%02d 11:00:00")
	barsByFreq := map[string][]any{"5m": long}
	pa := buildAnchoredPriceAnalysis(barsByFreq, map[string]bool{"5m": true}, long, "5m", anchor, 420, 428, KeyLevelBands{})
	pf := priceFilterForGateFrequency(pa, "5m", barsByFreq, long, anchor, "5m")
	if pf.BarCount > analysisBarsLimit("5m") {
		t.Fatalf("gate bar count=%d want ≤%d", pf.BarCount, analysisBarsLimit("5m"))
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
