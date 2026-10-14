package strategydev

import (
	"testing"
	"time"
)

func TestMultiTimeframeSpecsForGateUpwardOnly(t *testing.T) {
	cases := []struct {
		gate string
		want []string
	}{
		{"5m", []string{"5m", "60m", "daily", "weekly"}},
		{"60m", []string{"60m", "daily", "weekly"}},
		{"15m", []string{"60m", "daily", "weekly"}},
		{"daily", []string{"daily", "weekly"}},
		{"weekly", []string{"weekly"}},
		{"", []string{"5m", "60m", "daily", "weekly"}},
	}
	for _, tc := range cases {
		specs := multiTimeframeSpecsForGate(tc.gate)
		if len(specs) != len(tc.want) {
			t.Fatalf("gate=%q got %d specs want %d", tc.gate, len(specs), len(tc.want))
		}
		for i, w := range tc.want {
			if specs[i].Freq != w {
				t.Fatalf("gate=%q i=%d got=%s want=%s", tc.gate, i, specs[i].Freq, w)
			}
		}
	}
}

func TestBuildPriceAnalysisFromBars60mGateOmits5mRow(t *testing.T) {
	bars := map[string][]any{
		"5m":     buildTrendBars(90, "up"),
		"60m":    buildTrendBars(90, "up"),
		"daily":  buildTrendBars(50, "up"),
		"weekly": buildTrendBars(25, "down"),
	}
	pa := BuildPriceAnalysisFromBars(bars, 10, KeyLevelBands{}, "60m")
	if len(pa.Rows) != 3 {
		t.Fatalf("rows=%d want 3", len(pa.Rows))
	}
	for _, r := range pa.Rows {
		if r.Timeframe == "5m" {
			t.Fatal("60m gate must not include 5m row")
		}
	}
}

func TestBuildAnchoredPriceAnalysisDailyGateOmitsIntradayRows(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	anchor := time.Date(2026, 2, 1, 16, 0, 0, 0, loc)
	barsByFreq := map[string][]any{
		"daily":  {map[string]any{"time": "2026-03-01", "close": 1.0, "high": 1.1, "low": 0.9, "volume": 1.0}},
		"weekly": {map[string]any{"time": "2026-03-01", "close": 1.0, "high": 1.1, "low": 0.9, "volume": 1.0}},
	}
	fetchOK := map[string]bool{"daily": true, "weekly": true}
	pa := buildAnchoredPriceAnalysis(barsByFreq, fetchOK, nil, "daily", anchor, 100, 101, KeyLevelBands{})
	if len(pa.Rows) != 2 {
		t.Fatalf("rows=%d want 2", len(pa.Rows))
	}
	for _, r := range pa.Rows {
		if r.Timeframe == "5m" || r.Timeframe == "60m" {
			t.Fatalf("daily gate must not include intraday row %s", r.Timeframe)
		}
	}
}
