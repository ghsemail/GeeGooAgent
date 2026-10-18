package priceanalysis

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMonthsBackForLabFetchWeeklySignal3CoversAnalysisWindow(t *testing.T) {
	mb, err := MonthsBackForLabFetch(3, "weekly")
	if err != nil {
		t.Fatal(err)
	}
	// anchor 3 + analysis 6 + warmup 1
	if mb < 10 {
		t.Fatalf("weekly months_back=%d want >=10 for signal months=3", mb)
	}
	est := EstimateFetchBars("weekly", mb)
	if est < analysisBarsLimit("weekly") {
		t.Fatalf("estimated bars=%d want >= analysis limit %d", est, analysisBarsLimit("weekly"))
	}
	if est < minBarsForPriceFilter("weekly") {
		t.Fatalf("estimated bars=%d want >= min structure bars %d", est, minBarsForPriceFilter("weekly"))
	}
}

func TestMonthsBackPerTimeframeSignal3(t *testing.T) {
	cases := map[string]int{
		"5m":     4,  // 3+1+1 capped to ~5000 bars
		"60m":    5,  // 3+1+1
		"daily":  7,  // 3+3+1
		"weekly": 10, // 3+6+1
	}
	for freq, wantMin := range cases {
		mb, err := MonthsBackForLabFetch(3, freq)
		if err != nil {
			t.Fatalf("%s: %v", freq, err)
		}
		if mb < wantMin {
			t.Fatalf("%s months_back=%d want >= %d", freq, mb, wantMin)
		}
		if EstimateFetchBars(freq, mb) > LabKlineMaxBarsPerJob {
			t.Fatalf("%s exceeds bar cap", freq)
		}
	}
}

func Test00700HK5mMonths3WeeklyFetchMonthsRecorded(t *testing.T) {
	var gotWeekly int
	fetch := BarsFetcherFunc(func(_ context.Context, code, freq string, months int, _ *time.Time) ([]any, bool) {
		if code != "00700.HK" {
			t.Fatalf("code=%q", code)
		}
		if freq == "weekly" {
			gotWeekly = months
		}
		return synthBarsForFetchTest(freq, months), true
	})
	pa := Run(t.Context(), fetch, RunRequest{
		Code: "00700.HK", SignalFreq: "5m", Months: 3,
	})
	if gotWeekly < 10 {
		t.Fatalf("weekly months_back=%d want >=10", gotWeekly)
	}
	sc := ComputeMultiTFPriceScore(pa, "5m")
	if strings.Contains(sc.ScoreReason, "缺周期") && strings.Contains(sc.ScoreReason, "周线") {
		t.Fatalf("weekly still missing from composite: %q", sc.ScoreReason)
	}
	for _, r := range pa.Rows {
		if r.Timeframe == "周线" && r.Structure == structureInsufficientBars {
			t.Fatalf("weekly row insufficient: %+v", r)
		}
	}
}

func TestFormatLabPriceNoScientificNotation(t *testing.T) {
	cases := []float64{420, 4.2e2, 0.0088, 12345.678}
	for _, v := range cases {
		s := FormatLabPrice(v)
		if strings.ContainsAny(s, "eE") {
			t.Fatalf("FormatLabPrice(%v)=%q contains exponent", v, s)
		}
	}
	label := FormatLabSpotLabel(420)
	if strings.Contains(label, "e") {
		t.Fatalf("spot label=%q", label)
	}
	row := BuildPriceAnalysisRow(buildTrendBars(90, "up"), "60m", "60m")
	if strings.Contains(row.Structure, "e+") || strings.Contains(row.Structure, "e-") {
		t.Fatalf("structure=%q", row.Structure)
	}
}

// BarsFetcherFunc adapts a function to BarsFetcher for tests.
type BarsFetcherFunc func(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool)

func (f BarsFetcherFunc) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	return f(ctx, code, freq, months, asOf)
}

func synthBarsForFetchTest(freq string, monthsBack int) []any {
	n := EstimateFetchBars(freq, monthsBack)
	if n < minFetchBarsRequired(freq) {
		n = minFetchBarsRequired(freq)
	}
	if n > LabKlineMaxBarsPerJob {
		n = LabKlineMaxBarsPerJob
	}
	out := make([]any, n)
	for i := 0; i < n; i++ {
		c := 420.0 + float64(i)*0.01
		out[i] = map[string]any{
			"time": "2026-01-01 10:00:00", "open": c, "high": c + 1, "low": c - 1, "close": c, "volume": 1000.0,
		}
	}
	return out
}
