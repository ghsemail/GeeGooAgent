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

func TestEvaluateSignalEpisodes_SellUntilBuy(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0},
		map[string]any{"time": "t1", "close": 95.0},
		map[string]any{"time": "t2", "close": 90.0},
		map[string]any{"time": "t3", "close": 92.0},
		map[string]any{"time": "t4", "close": 88.0},
	}
	buyMerged := []any{0, 0, 0, 1, 0}
	sellMerged := []any{0, -1, 0, 0, 0}
	eval := EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if eval.SellEpisodes.CompleteCount != 1 {
		t.Fatalf("sell complete=%d want 1", eval.SellEpisodes.CompleteCount)
	}
	if len(eval.SellDetails) != 1 || !eval.SellDetails[0].Hit {
		t.Fatalf("sell detail=%+v want hit (price down)", eval.SellDetails[0])
	}
	if eval.SellDetails[0].HoldingBars != 1 {
		t.Fatalf("holding=%d want 1 (t1->t2)", eval.SellDetails[0].HoldingBars)
	}
}

func TestEvaluateSignalEpisodes_IncompleteTail(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0, "high": 100.0, "low": 100.0},
		map[string]any{"time": "t1", "close": 101.0, "high": 102.0, "low": 99.5},
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
	if eval.BuyPathEpisodes.CompleteCount != 1 {
		t.Fatalf("path complete=%d want 1", eval.BuyPathEpisodes.CompleteCount)
	}
	if len(eval.BuyDetails) != 1 || !eval.BuyDetails[0].PathEvaluable {
		t.Fatalf("path evaluable=%+v", eval.BuyDetails[0])
	}
	if eval.BuyDetails[0].PathEndReason != pathEndOpenTail {
		t.Fatalf("reason=%q", eval.BuyDetails[0].PathEndReason)
	}
}

func TestEvaluateSignalEpisodes_PathUntilNextBuy(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 100.0, "high": 100.0, "low": 100.0},
		map[string]any{"time": "t1", "close": 110.0, "high": 112.0, "low": 99.0},
		map[string]any{"time": "t2", "close": 105.0, "high": 106.0, "low": 104.0},
		map[string]any{"time": "t3", "close": 108.0, "high": 109.0, "low": 107.0},
	}
	buyMerged := []any{1, 0, 0, 1}
	sellMerged := []any{0, 0, 0, 0}
	eval := EvaluateSignalEpisodes(bars, buyMerged, sellMerged)
	if eval.BuyEpisodes.IncompleteCount != 2 {
		t.Fatalf("strict incomplete=%d want 2", eval.BuyEpisodes.IncompleteCount)
	}
	if eval.BuyPathEpisodes.CompleteCount != 2 {
		t.Fatalf("path count=%d want 2", eval.BuyPathEpisodes.CompleteCount)
	}
	first := eval.BuyDetails[0]
	if first.PathEndReason != pathEndNextSame || first.PathEndIdx != 2 {
		t.Fatalf("first path=%+v", first)
	}
	if first.PeakReturn <= 0 {
		t.Fatalf("peak return=%v", first.PeakReturn)
	}
	if first.MaxDrawdownFromPeak <= 0 {
		t.Fatalf("max dd=%v", first.MaxDrawdownFromPeak)
	}
}
