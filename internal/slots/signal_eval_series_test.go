package slots

import "testing"

func TestEvaluateSignalEpisodes_KeyBreakPerBarSeries(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0, "high": 101.0, "low": 99.0},
		map[string]any{"time": "t1", "close": 100.0, "high": 101.0, "low": 97.0},
		map[string]any{"time": "t2", "close": 102.0, "high": 103.0, "low": 101.0},
	}
	buyMerged := []any{1, 0, 0}
	sellMerged := []any{0, 0, 0}
	stop := &KeyLevelEpisodeStop{
		BuyBreakRef:  "support_low",
		SellBreakRef: "resist_high",
		SupportLowSeries: []float64{99.5, 98.0, 98.0},
	}
	eval := EvaluateSignalEpisodesWithKeyStop(bars, buyMerged, sellMerged, stop)
	if eval.BuyEpisodes.CompleteCount != 1 {
		t.Fatalf("complete=%d want 1", eval.BuyEpisodes.CompleteCount)
	}
	d := eval.BuyDetails[0]
	if d.EndIdx != 1 {
		t.Fatalf("end=%d want 1 (bar1 low 97 < support 98)", d.EndIdx)
	}
}
