package klinepattern

import "testing"

func TestPatternHitsForDisplayInput_FallsBackToMTFSignalSeries(t *testing.T) {
	mtf := &KlinePatternMTFBundle{
		SignalFreq: "60m",
		ByFreq: map[string]KlinePatternFreqSeries{
			klinePatternMTFTimeframeKey("60m"): {
				Freq:        "60m",
				PatternHits: []any{nil, []any{map[string]any{"name": "Hammer", "pattern": 1}}},
			},
		},
	}
	in := NewKlinePatternDisplayInput("60m", []any{
		map[string]any{"close": 1},
		map[string]any{"close": 2},
	}, map[string]any{}, mtf)
	if len(in.PatternHits) != 2 {
		t.Fatalf("pattern_hits=%d", len(in.PatternHits))
	}
}
