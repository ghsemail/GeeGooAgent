package klinepattern

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestFilteredKlinePatternsAtIndex_GeeGooPatternField(t *testing.T) {
	series := []any{
		[]any{map[string]any{"name": "Engulfing", "name_chs": "吞没模式", "pattern": -1}},
		[]any{map[string]any{"name": "Evening Star", "pattern": 1}},
	}
	h0 := FilteredKlinePatternsAtIndex(series, 0)
	if len(h0) != 1 || h0[0].Direction != -1 {
		t.Fatalf("engulfing=%+v", h0)
	}
	b0 := slots.ScoreKlinePatternHits(h0)
	if b0.Score == 0 || b0.Bias == "中性" {
		t.Fatalf("score=%+v want non-neutral bear", b0)
	}
	h1 := FilteredKlinePatternsAtIndex(series, 1)
	if len(h1) != 1 || h1[0].Name != "EveningStar" || h1[0].Direction != 1 {
		t.Fatalf("evening star=%+v", h1)
	}
	if slots.ScoreKlinePatternHits(h1).Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestFilteredKlinePatternsAtIndex_WhitelistAndDirection(t *testing.T) {
	series := []any{
		[]any{map[string]any{"name": "Hammer", "direction": 1}},
		[]any{
			map[string]any{"name": "Engulfing", "direction": -1},
			map[string]any{"name": "CDLMORNINGSTAR", "direction": 1},
			map[string]any{"name": "RandomPattern", "direction": 1},
		},
	}
	h0 := FilteredKlinePatternsAtIndex(series, 0)
	if len(h0) != 1 || h0[0].Name != "Hammer" || h0[0].Direction != 1 {
		t.Fatalf("bar0=%+v", h0)
	}
	h1 := FilteredKlinePatternsAtIndex(series, 1)
	if len(h1) != 2 {
		t.Fatalf("bar1 count=%d %+v", len(h1), h1)
	}
	if h1[0].Name != "Engulfing" || h1[1].Name != "MorningStar" {
		t.Fatalf("names=%+v", h1)
	}
}

func TestKlinePatternSummary_MixedDirections(t *testing.T) {
	s := slots.KlinePatternSummary([]slots.KlinePatternHit{
		{Name: "Engulfing", Direction: 1, LabelCN: "吞没形态"},
		{Name: "ShootingStar", Direction: -1, LabelCN: "射击之星"},
	})
	if s == "" || s[:len("多空并存：")] != "多空并存：" {
		t.Fatalf("summary=%q", s)
	}
}

func TestEnrichSignalEpisodeEvalPatterns_AttachesAtStartIdx(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "close": 1},
		map[string]any{"time": "2024-01-02", "close": 2},
	}
	buyM := []any{1, 0}
	sellM := []any{0, 0}
	eval := slots.EvaluateSignalEpisodes(bars, buyM, sellM)
	series := []any{
		nil,
		[]any{map[string]any{"name": "Doji", "direction": 2}},
	}
	patIn := NewKlinePatternDisplayInput("daily", bars, map[string]any{"pattern_hits": series}, nil)
	EnrichSignalEpisodeEvalKlineDisplay(&eval, patIn)
	if len(eval.BuyDetails) != 1 {
		t.Fatalf("buy details=%d", len(eval.BuyDetails))
	}
	if len(eval.BuyDetails[0].KlinePatterns) != 0 {
		t.Fatalf("expected patterns at start idx 0 empty, got %+v", eval.BuyDetails[0].KlinePatterns)
	}
	series[0] = []any{map[string]any{"name": "Hammer", "direction": 1}}
	patIn.PatternHits = series
	EnrichSignalEpisodeEvalKlineDisplay(&eval, patIn)
	if len(eval.BuyDetails[0].KlinePatterns) != 1 || eval.BuyDetails[0].KlinePatternSummary == "" {
		t.Fatalf("detail=%+v", eval.BuyDetails[0])
	}
	if eval.BuyDetails[0].KlinePatternThreeBar == nil || eval.BuyDetails[0].KlinePatternBias == nil {
		t.Fatal("expected three-bar window and bias")
	}
}

func TestEvaluateSignalEpisodes_UnchangedByPatternEnrichment(t *testing.T) {
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
		map[string]any{"time": "2024-01-03", "open": 2, "high": 2.2, "low": 1.5, "close": 1.6},
	}
	buyM := []any{1, 0, 0}
	sellM := []any{0, 0, -1}
	before := slots.EvaluateSignalEpisodes(bars, buyM, sellM)
	after := slots.EvaluateSignalEpisodes(bars, buyM, sellM)
	series := []any{
		[]any{map[string]any{"name": "Hammer", "direction": 1}},
		nil,
		[]any{map[string]any{"name": "EveningStar", "direction": -1}},
	}
	EnrichSignalEpisodeEvalKlineDisplay(&after, NewKlinePatternDisplayInput("daily", bars, map[string]any{"pattern_hits": series}, nil))
	if before.BuyEpisodes.HitCount != after.BuyEpisodes.HitCount {
		t.Fatalf("hit count changed: before=%d after=%d", before.BuyEpisodes.HitCount, after.BuyEpisodes.HitCount)
	}
	if before.BuyDetails[0].Hit != after.BuyDetails[0].Hit {
		t.Fatalf("episode hit changed")
	}
}

