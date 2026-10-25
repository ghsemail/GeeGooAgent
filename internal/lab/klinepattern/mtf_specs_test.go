package klinepattern

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestMultiTimeframeSpecsForKlinePatternDisplay_Ladders(t *testing.T) {
	cases := []struct {
		sig  string
		want []string
	}{
		{"5m", []string{"5m", "60m", "daily", "weekly"}},
		{"5分钟", []string{"5m", "60m", "daily", "weekly"}},
		{"60m", []string{"60m", "daily", "weekly"}},
		{"1h", []string{"60m", "daily", "weekly"}},
		{"daily", []string{"daily", "weekly"}},
		{"日线", []string{"daily", "weekly"}},
		{"weekly", []string{"weekly"}},
		{"1wk", []string{"weekly"}},
		{"15m", []string{"60m", "daily", "weekly"}},
		{"", []string{"5m", "60m", "daily", "weekly"}},
	}
	for _, tc := range cases {
		specs := multiTimeframeSpecsForKlinePatternDisplay(tc.sig)
		wantLen := len(tc.want)
		if len(specs) != wantLen {
			t.Fatalf("sig=%q got %d specs want %d", tc.sig, len(specs), wantLen)
		}
		for i, w := range tc.want {
			if specs[i].Freq != w {
				t.Fatalf("sig=%q i=%d got=%s want=%s", tc.sig, i, specs[i].Freq, w)
			}
		}
	}
}

func TestKlinePatternMTFWeights_TableLocked(t *testing.T) {
	if len(KlinePatternMTFWeights) != 4 {
		t.Fatalf("weights len=%d", len(KlinePatternMTFWeights))
	}
	var sum float64
	for i, w := range KlinePatternMTFWeights {
		sum += w
		switch i {
		case 0:
			if w != 0.4 {
				t.Fatalf("w0=%v", w)
			}
		case 1:
			if w != 0.3 {
				t.Fatalf("w1=%v", w)
			}
		case 2:
			if w != 0.2 {
				t.Fatalf("w2=%v", w)
			}
		case 3:
			if w != 0.1 {
				t.Fatalf("w3=%v", w)
			}
		}
	}
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("sum=%v", sum)
	}
}

func TestKlinePatternMTFWeights_RenormalizedForShorterLadder(t *testing.T) {
	w3 := klinePatternMTFWeightsForLadder(3)
	if len(w3) != 3 {
		t.Fatalf("len=%d", len(w3))
	}
	var sum float64
	for _, w := range w3 {
		sum += w
	}
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("sum=%v", sum)
	}
	if w3[0] <= w3[1] || w3[1] <= w3[2] {
		t.Fatalf("signal should stay heaviest: %v", w3)
	}

	w1 := klinePatternMTFWeightsForLadder(1)
	if len(w1) != 1 || w1[0] != 1 {
		t.Fatalf("single TF weight=%v", w1)
	}
}

func TestKlinePatternProbeFrequency_MatchesNormalize(t *testing.T) {
	if got := klinePatternProbeFrequency("daily"); got != "daily" {
		t.Fatalf("daily probe=%q", got)
	}
	if NormalizeFrequency("60m") != "60m" {
		t.Fatal("60m")
	}
	if NormalizeFrequency("1h") != "60m" {
		t.Fatal("1h -> 60m")
	}
}

func TestBuildKlinePatternMTFContext_FiveMinuteLadder(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
	}
	hits := []any{nil, []any{map[string]any{"name": "Hammer", "direction": 1}}}
	in := KlinePatternDisplayInput{
		SignalFreq:  "5m",
		Bars:        bars,
		PatternHits: hits,
		MTF: &KlinePatternMTFBundle{
			SignalFreq: "5m",
			ByFreq: map[string]KlinePatternFreqSeries{
				"60m":    {Freq: "60m", FetchOK: false},
				"daily":  {Freq: "daily", FetchOK: false},
				"weekly": {Freq: "weekly", FetchOK: false},
			},
		},
	}
	ctx := BuildKlinePatternMTFContext(1, in)
	if ctx == nil {
		t.Fatal("nil context")
	}
	var bias *slots.KlinePatternBias
	ApplyMTFWeightedKlineConclusion(ctx, &bias, nil)
	wantFreqs := []string{"5m", "60m", "daily", "weekly"}
	if len(ctx.Timeframes) != len(wantFreqs) {
		t.Fatalf("timeframes=%d want %d", len(ctx.Timeframes), len(wantFreqs))
	}
	for i, w := range wantFreqs {
		if ctx.Timeframes[i].Frequency != w {
			t.Fatalf("i=%d freq=%s want=%s", i, ctx.Timeframes[i].Frequency, w)
		}
		if !ctx.Timeframes[i].DisplayOnly {
			t.Fatalf("i=%d display_only=false", i)
		}
	}
	if ctx.Timeframes[0].ThreeBar == nil || len(ctx.Timeframes[0].ThreeBar.Bars) != 3 {
		t.Fatalf("signal three_bar=%+v", ctx.Timeframes[0].ThreeBar)
	}
	roles := []string{
		ctx.Timeframes[0].ThreeBar.Bars[0].Role,
		ctx.Timeframes[0].ThreeBar.Bars[1].Role,
		ctx.Timeframes[0].ThreeBar.Bars[2].Role,
	}
	if roles[0] != "prev2" || roles[1] != "prev" || roles[2] != "anchor" {
		t.Fatalf("three_bar roles=%v", roles)
	}
	if ctx.WeightSchema != KlinePatternMTFWeightSchema {
		t.Fatalf("weight_schema=%q", ctx.WeightSchema)
	}
	if ctx.PrimaryBias == nil {
		t.Fatal("expected primary_bias from weighted conclusion")
	}
}

func TestBuildKlinePatternMTFContext_WeeklySingleRow(t *testing.T) {
	bars := []any{map[string]any{"time": "2024-01-01", "close": 1}}
	in := KlinePatternDisplayInput{
		SignalFreq:  "weekly",
		Bars:        bars,
		PatternHits: []any{nil},
		MTF:         &KlinePatternMTFBundle{SignalFreq: "weekly", ByFreq: map[string]KlinePatternFreqSeries{}},
	}
	ctx := BuildKlinePatternMTFContext(0, in)
	if ctx == nil || len(ctx.Timeframes) != 1 || ctx.Timeframes[0].Frequency != "weekly" {
		t.Fatalf("ctx=%+v", ctx)
	}
}
