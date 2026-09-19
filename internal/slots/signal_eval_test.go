package slots

import "testing"

func TestEvaluateSignalEpisodes_BuyUntilSell(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0},
		map[string]any{"time": "t1", "close": 110.0},
		map[string]any{"time": "t2", "close": 105.0},
		map[string]any{"time": "t3", "close": 95.0},
		map[string]any{"time": "t4", "close": 90.0},
	}
	buyMerged := []any{1, 0, 0, 0, 0}
	sellMerged := []any{0, 0, 0, -1, 0}
	eval := EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if eval.BuyEpisodes.CompleteCount != 1 {
		t.Fatalf("buy complete=%d want 1", eval.BuyEpisodes.CompleteCount)
	}
	if eval.BuyEpisodes.HitCount != 1 {
		t.Fatalf("buy hit=%d want 1 (100@t0 -> 105@t2)", eval.BuyEpisodes.HitCount)
	}
	if eval.SellEpisodes.CompleteCount != 0 {
		t.Fatalf("sell complete=%d want 0 (no sell start before buy)", eval.SellEpisodes.CompleteCount)
	}
}

func TestEvaluateSignalEpisodes_MergesConsecutiveBuys(t *testing.T) {
	bars := []any{
		map[string]any{"close": 100.0},
		map[string]any{"close": 101.0},
		map[string]any{"close": 102.0},
		map[string]any{"close": 98.0},
	}
	buyMerged := []any{0, 1, 1, 0}
	sellMerged := []any{0, 0, 0, -1}
	eval := EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if eval.BuyEpisodes.CompleteCount != 1 {
		t.Fatalf("complete=%d want 1", eval.BuyEpisodes.CompleteCount)
	}
	if eval.BuyEpisodes.HitCount != 1 {
		t.Fatalf("hit=%d want 1 (101@t1 -> 102@t2 before sell)", eval.BuyEpisodes.HitCount)
	}
}

func TestEvaluateSignalEpisodes_IncompleteTail(t *testing.T) {
	bars := []any{
		map[string]any{"close": 100.0},
		map[string]any{"close": 101.0},
	}
	buyMerged := []any{0, 1}
	sellMerged := []any{0, 0}
	eval := EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if eval.BuyEpisodes.CompleteCount != 0 {
		t.Fatalf("complete=%d want 0", eval.BuyEpisodes.CompleteCount)
	}
	if eval.BuyEpisodes.IncompleteCount != 1 {
		t.Fatalf("incomplete=%d want 1", eval.BuyEpisodes.IncompleteCount)
	}
}
