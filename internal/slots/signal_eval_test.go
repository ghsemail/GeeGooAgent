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
	if len(eval.BuyDetails) != 1 || !eval.BuyDetails[0].Hit {
		t.Fatalf("buy detail=%+v want hit", eval.BuyDetails[0])
	}
	if eval.BuyDetails[0].HoldingBars != 2 {
		t.Fatalf("holding=%d want 2 (t0->t2)", eval.BuyDetails[0].HoldingBars)
	}
}

func TestEvaluateSignalEpisodes_MergesConsecutiveBuys(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0},
		map[string]any{"time": "t1", "close": 101.0},
		map[string]any{"time": "t2", "close": 99.0},
		map[string]any{"time": "t3", "close": 98.0},
	}
	buyMerged := []any{0, 1, 1, 0}
	sellMerged := []any{0, 0, 0, -1}
	eval := EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if len(eval.BuyDetails) != 1 {
		t.Fatalf("details=%d want 1 merged episode", len(eval.BuyDetails))
	}
	if eval.BuyDetails[0].Hit {
		t.Fatalf("expected miss 101->99 before sell")
	}
}

func TestEvaluateSignalEpisodes_IncompleteTail(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0},
		map[string]any{"time": "t1", "close": 101.0},
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
