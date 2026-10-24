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
		{"5m", []string{"5m", "15m", "30m", "60m"}},
		{"5分钟", []string{"5m", "15m", "30m", "60m"}},
		{"60m", []string{"60m", "daily", "weekly"}}, // freq keys; display labels use 1小时
		{"1h", []string{"60m", "daily", "weekly"}},
		{"daily", []string{"daily", "weekly"}},
		{"日线", []string{"daily", "weekly"}},
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

func TestBuildKlinePatternMTFContext_AlwaysFourTimeframes(t *testing.T) {
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
				"15m":   {Freq: "15m", FetchOK: false},
				"30m":   {Freq: "30m", FetchOK: false},
				"60m":   {Freq: "60m", FetchOK: false},
			},
		},
	}
	ctx := BuildKlinePatternMTFContext(1, in)
	if ctx == nil {
		t.Fatal("nil context")
	}
	var bias *slots.KlinePatternBias
	ApplyMTFWeightedKlineConclusion(ctx, &bias, nil)
	if len(ctx.Timeframes) != 4 {
		t.Fatalf("timeframes=%d want 4", len(ctx.Timeframes))
	}
	wantFreqs := []string{"5m", "15m", "30m", "60m"}
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
