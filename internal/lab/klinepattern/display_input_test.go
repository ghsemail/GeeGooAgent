package klinepattern

import "testing"

func TestPatternHitsForDisplayInput_FallsBackToMTFSignalSeries(t *testing.T) {
	mtf := &KlinePatternMTFBundle{SignalFreq: "60m", ByFreq: map[string]KlinePatternFreqSeries{}}
	chartBars := []any{
		map[string]any{"time": "2024-01-02", "close": 2},
	}
	mtf.ByFreq[klinePatternMTFTimeframeKey("60m")] = KlinePatternFreqSeries{
		Freq: "60m",
		Bars: []any{
			map[string]any{"time": "2024-01-01", "close": 1},
			map[string]any{"time": "2024-01-02", "close": 2},
		},
		PatternHits: []any{
			nil,
			[]any{map[string]any{"name": "Hammer", "pattern": 1}},
		},
	}
	in := NewKlinePatternDisplayInput("60m", chartBars, map[string]any{}, mtf)
	if len(in.PatternHits) != 1 {
		t.Fatalf("pattern_hits=%d", len(in.PatternHits))
	}
	h := FilteredKlinePatternsAtIndex(in.PatternHits, 0)
	if len(h) != 1 || h[0].Name != "Hammer" {
		t.Fatalf("hits=%+v", h)
	}
}
