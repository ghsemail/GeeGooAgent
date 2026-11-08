package klinepattern

import "testing"

func TestPatternHitsForDisplayInput_DailyMacroPrefersMTFDailySeries(t *testing.T) {
	mtf := &KlinePatternMTFBundle{SignalFreq: "daily", ByFreq: map[string]KlinePatternFreqSeries{}}
	chartBars := []any{
		map[string]any{"time": "2024-01-02T12:00:00Z", "close": 2},
	}
	mtf.ByFreq[klinePatternMTFTimeframeKey("daily")] = KlinePatternFreqSeries{
		Freq:  "daily",
		Label: "日线",
		Bars: []any{
			map[string]any{"time": "2024-01-01", "close": 1},
			map[string]any{"time": "2024-01-02", "close": 2},
		},
		PatternHits: []any{
			nil,
			[]any{map[string]any{"name": "Engulfing", "pattern": 1}},
		},
		FetchOK: true,
	}
	probe := map[string]any{
		"pattern_hits": []any{
			[]any{map[string]any{"name": "Hammer", "pattern": 1}},
		},
	}
	in := NewKlinePatternDisplayInput("daily", chartBars, probe, mtf)
	h := FilteredKlinePatternsAtIndex(in.PatternHits, 0)
	if len(h) != 1 || h[0].Name != "Engulfing" {
		t.Fatalf("want daily MTF hit, got %+v", h)
	}
	ctx := BuildKlinePatternMTFContext(0, in)
	if ctx == nil || len(ctx.Timeframes) == 0 {
		t.Fatal("expected mtf context")
	}
	tf := ctx.Timeframes[0]
	if tf.ThreeBar == nil {
		t.Fatal("missing three bar")
	}
	anchorHits := FilteredKlinePatternsAtIndex([]any{tf.ThreeBar.Bars[1].KlinePatterns}, 0)
	if len(anchorHits) == 0 {
		for _, b := range tf.ThreeBar.Bars {
			if b.Role == "anchor" && len(b.KlinePatterns) > 0 {
				anchorHits = b.KlinePatterns
				break
			}
		}
	}
	if len(anchorHits) == 0 || anchorHits[0].Name != "Engulfing" {
		t.Fatalf("anchor patterns=%+v", anchorHits)
	}
}
