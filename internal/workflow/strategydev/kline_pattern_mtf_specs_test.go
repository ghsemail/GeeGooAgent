package strategydev

import "testing"

func TestMultiTimeframeSpecsForKlinePatternDisplay_FiveAndSixtyMinute(t *testing.T) {
	cases := []struct {
		sig  string
		want []string
	}{
		{"5m", []string{"5m", "30m", "60m", "daily"}},
		{"5分钟", []string{"5m", "30m", "60m", "daily"}},
		{"60m", []string{"60m", "30m", "daily", "weekly"}},
		{"1h", []string{"60m", "30m", "daily", "weekly"}},
	}
	for _, tc := range cases {
		specs := multiTimeframeSpecsForKlinePatternDisplay(tc.sig)
		if len(specs) != 4 {
			t.Fatalf("sig=%q got %d specs want 4", tc.sig, len(specs))
		}
		for i, w := range tc.want {
			if specs[i].Freq != w {
				t.Fatalf("sig=%q i=%d got=%s want=%s", tc.sig, i, specs[i].Freq, w)
			}
		}
	}
}

func TestBuildKlinePatternMTFContext_AlwaysFourTimeframes(t *testing.T) {
	SetKlinePatternDisplayEnabledForTest(t, true)
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
				"30m":   {Freq: "30m", FetchOK: false},
				"60m":   {Freq: "60m", FetchOK: false},
				"daily": {Freq: "daily", FetchOK: false},
			},
		},
	}
	ctx := buildKlinePatternMTFContext(1, in)
	if ctx == nil {
		t.Fatal("nil context")
	}
	if len(ctx.Timeframes) != 4 {
		t.Fatalf("timeframes=%d want 4", len(ctx.Timeframes))
	}
	wantFreqs := []string{"5m", "30m", "60m", "daily"}
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
	if ctx.PrimaryBias == nil {
		t.Fatal("expected primary_bias from signal anchor")
	}
}
