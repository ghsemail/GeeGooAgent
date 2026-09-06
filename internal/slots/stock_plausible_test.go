package slots

import "testing"

func TestStockQueryPlausible(t *testing.T) {
	cases := []struct {
		q    string
		want bool
	}{
		{"中际旭创", true},
		{"300308", true},
		{"00700", true},
		{"AAPL", true},
		{"小米", true},
		{"卖点", false},
		{"买点", false},
		{"有没有", false},
		{"MACD", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := StockQueryPlausible(tc.q); got != tc.want {
			t.Fatalf("StockQueryPlausible(%q)=%v want %v", tc.q, got, tc.want)
		}
	}
}

func TestExtractStockQueryStillReturnsMaiDianWithoutPlausible(t *testing.T) {
	// Heuristic extractor may still emit junk; planners must gate with StockQueryPlausible.
	msg := "就用SAR加MACD组合，帮我测一下中际旭创有没有买卖点"
	got := ExtractStockQuery(msg)
	if got == "中际旭创" {
		return
	}
	if !StockQueryPlausible(got) {
		return
	}
	t.Fatalf("ExtractStockQuery(%q)=%q should not be a plausible ticker", msg, got)
}
