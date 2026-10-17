package strategydev

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestBuildKlinePatternThreeBarWindow_Boundaries(t *testing.T) {
	SetKlinePatternDisplayEnabledForTest(t, true)
	bars := []any{
		map[string]any{"time": "2024-01-01", "open": 1, "high": 2, "low": 0.5, "close": 1.5},
		map[string]any{"time": "2024-01-02", "open": 1.5, "high": 2.5, "low": 1, "close": 2},
	}
	hits := []any{
		nil,
		[]any{map[string]any{"name": "Hammer", "direction": 1}},
	}
	win := BuildKlinePatternThreeBarWindow(bars, hits, 0)
	if win == nil || len(win.Bars) != 3 {
		t.Fatalf("win=%+v", win)
	}
	if !win.Bars[0].Omitted || win.Bars[0].Role != "prev" {
		t.Fatalf("prev=%+v", win.Bars[0])
	}
	if win.Bars[1].Role != "anchor" || win.Bars[1].Close != 1.5 {
		t.Fatalf("anchor=%+v", win.Bars[1])
	}
	if win.Bars[2].Omitted || win.Bars[2].Index != 1 {
		t.Fatalf("next=%+v", win.Bars[2])
	}
	if win.WindowNote == "" {
		t.Fatal("expected window note for missing prev")
	}
	winLast := BuildKlinePatternThreeBarWindow(bars, hits, 1)
	if winLast == nil || !winLast.Bars[2].Omitted {
		t.Fatalf("last anchor next=%+v", winLast.Bars[2])
	}
}

func TestBuildKlinePatternThreeBarWindow_Scoring(t *testing.T) {
	SetKlinePatternDisplayEnabledForTest(t, true)
	bars := []any{
		map[string]any{"time": "2024-01-01", "close": 1},
		map[string]any{"time": "2024-01-02", "close": 2},
		map[string]any{"time": "2024-01-03", "close": 3},
	}
	hits := []any{
		[]any{map[string]any{"name": "Hammer", "direction": 1}},
		[]any{
			map[string]any{"name": "Engulfing", "direction": 1},
			map[string]any{"name": "ShootingStar", "direction": -1},
		},
		nil,
	}
	win := BuildKlinePatternThreeBarWindow(bars, hits, 1)
	if win.Bias == nil || win.Bias.Score != 0 {
		t.Fatalf("anchor bias=%+v want neutral mix", win.Bias)
	}
	if win.WindowBias == nil || win.WindowBias.Score != 1 {
		t.Fatalf("window bias=%+v want +1", win.WindowBias)
	}
	if !containsStr(win.Bars[1].KlinePatternSummary, "多空并存") {
		t.Fatalf("summary=%q", win.Bars[1].KlinePatternSummary)
	}
}

func TestScoreKlinePatternHits(t *testing.T) {
	b := slots.ScoreKlinePatternHits([]slots.KlinePatternHit{{Direction: 1}, {Direction: -1}})
	if b.Bias != "中性" || b.BiasColor != "gray" {
		t.Fatalf("got %+v", b)
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && stringIndex(s, sub)))
}

func stringIndex(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
