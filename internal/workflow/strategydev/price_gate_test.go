package strategydev

import (
	"testing"
	"time"
)

func TestHardGateParticipatingFrequencies60mExcludes5m(t *testing.T) {
	got := hardGateParticipatingFrequencies("60m")
	want := []string{"60m", "daily", "weekly"}
	if len(got) != len(want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}

func TestHardGateParticipatingFrequenciesDailyExcludesIntraday(t *testing.T) {
	got := hardGateParticipatingFrequencies("daily")
	want := []string{"daily", "weekly"}
	if len(got) != len(want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}

func TestShouldInterceptSellOnlyOnGateUp(t *testing.T) {
	cases := []struct {
		regime   string
		intercept bool
	}{
		{RegimeUp, true},
		{RegimeDown, false},
		{RegimeRange, false},
		{RegimeUnclear, false},
		{RegimeWeakUp, false},
	}
	for _, tc := range cases {
		pf := PriceFilterResult{Regime: tc.regime, AllowLong: false, AllowShort: tc.regime != RegimeUp}
		if got := shouldInterceptTrigger("sell", pf); got != tc.intercept {
			t.Fatalf("regime=%s intercept=%v want=%v", tc.regime, got, tc.intercept)
		}
	}
}

func TestPriceFilterForGate60mIgnores5mRowForBuyAllowLong(t *testing.T) {
	// No bars: gate uses row conclusions; 5m row is bearish but must not enter hard gate at 60m.
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "5m", Conclusion: RegimeDown},
			{Timeframe: "60m", Conclusion: RegimeUp},
			{Timeframe: "日线", Conclusion: RegimeUp},
			{Timeframe: "周线", Conclusion: RegimeUp},
		},
	}
	pf := priceFilterForGateFrequency(pa, "60m", map[string][]any{}, nil, time.Time{}, "60m")
	if pf.Regime != RegimeUp {
		t.Fatalf("gate regime=%s", pf.Regime)
	}
	if !pf.AllowLong {
		t.Fatal("5m down must not block buy when 60m+ allow long")
	}
	if shouldInterceptTrigger("buy", pf) {
		t.Fatal("buy should not intercept")
	}
	if !shouldInterceptTrigger("sell", pf) {
		t.Fatal("sell should intercept when gate regime is uptrend")
	}
}

func TestPriceFilterForGate60mComposedAllowLongLegacyOnly(t *testing.T) {
	pa := PriceAnalysis{
		Rows: []PriceAnalysisRow{
			{Timeframe: "5m", Conclusion: RegimeDown},
			{Timeframe: "60m", Conclusion: RegimeUp},
			{Timeframe: "日线", Conclusion: RegimeUnclear},
			{Timeframe: "周线", Conclusion: RegimeUp},
		},
	}
	pf := priceFilterForGateFrequency(pa, "60m", map[string][]any{}, nil, time.Time{}, "60m")
	if pf.AllowLong {
		t.Fatal("daily unclear should block composed allow_long")
	}
	sc := ComputeMultiTFPriceScore(pa, "60m")
	if ShouldInterceptTriggerByScore("buy", sc.Score, 0) {
		t.Fatalf("score gate must not mirror legacy allow_long block; score=%.1f", sc.Score)
	}
}
