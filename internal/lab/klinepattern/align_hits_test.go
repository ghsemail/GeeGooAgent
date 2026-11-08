package klinepattern

import "testing"

func TestAlignPatternHitsSeriesToBars_ByTime(t *testing.T) {
	probeBars := []any{
		map[string]any{"time": "2024-01-01T00:00:00Z", "close": 1},
		map[string]any{"time": "2024-01-02T00:00:00Z", "close": 2},
		map[string]any{"time": "2024-01-03T00:00:00Z", "close": 3},
	}
	hits := []any{
		nil,
		[]any{map[string]any{"name": "Hammer", "pattern": 1}},
		[]any{map[string]any{"name": "Doji", "pattern": 2}},
	}
	chartBars := []any{
		map[string]any{"time": "2024-01-02T00:00:00Z", "close": 2},
		map[string]any{"time": "2024-01-03T00:00:00Z", "close": 3},
	}
	aligned := AlignPatternHitsSeriesToBars(chartBars, probeBars, hits)
	if len(aligned) != 2 {
		t.Fatalf("aligned len=%d", len(aligned))
	}
	h0 := FilteredKlinePatternsAtIndex(aligned, 0)
	if len(h0) != 1 || h0[0].Name != "Hammer" {
		t.Fatalf("bar0=%+v", h0)
	}
}
