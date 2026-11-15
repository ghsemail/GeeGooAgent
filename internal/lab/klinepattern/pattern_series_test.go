package klinepattern

import "testing"

func TestPatternHitsSeriesPopulated(t *testing.T) {
	empty := []any{[]any{}, []any{}}
	if patternHitsSeriesPopulated(empty) {
		t.Fatal("expected false for all-empty slots")
	}
	hit := []any{[]any{map[string]any{"name": "Doji", "pattern": 2}}}
	if !patternHitsSeriesPopulated(hit) {
		t.Fatal("expected true when one bar has hits")
	}
}

func TestBarSeriesSameTimes(t *testing.T) {
	barsA := []any{
		map[string]any{"time": "2026-11-14T00:00:00Z", "close": 1},
		map[string]any{"time": "2026-11-15T00:00:00Z", "close": 2},
	}
	barsB := []any{
		map[string]any{"time": "2026-11-13T00:00:00Z", "close": 1},
		map[string]any{"time": "2026-11-14T00:00:00Z", "close": 2},
	}
	if barSeriesSameTimes(barsA, barsB) {
		t.Fatal("different calendar days should not match")
	}
	if !barSeriesSameTimes(barsA, barsA) {
		t.Fatal("identical series should match")
	}
}