package tools

import "testing"

func TestMatchClarifyAnswerLetter(t *testing.T) {
	choices := []string{"00700.HK 腾讯控股", "01698.HK 腾讯音乐-SW"}
	got, ok := MatchClarifyAnswer("A", choices)
	if !ok || got != choices[0] {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestMatchClarifyAnswerIndex(t *testing.T) {
	choices := []string{"a", "b", "c"}
	got, ok := MatchClarifyAnswer("2", choices)
	if !ok || got != "b" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestMatchClarifyAnswerChineseOrdinal(t *testing.T) {
	choices := []string{"日线", "周线"}
	got, ok := MatchClarifyAnswer("第二", choices)
	if !ok || got != "周线" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}
