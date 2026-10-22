package slots

import "testing"

func TestPickCombinationTieBreakBySignalID(t *testing.T) {
	items := []map[string]any{
		{"name": "SAR MACD A", "signal_id": "z-last", "buy_signal": []any{map[string]any{"index": "Z"}}},
		{"name": "SAR MACD B", "signal_id": "a-first", "buy_signal": []any{map[string]any{"index": "A"}}},
	}
	row, err := PickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "a-first" {
		t.Fatalf("expected deterministic a-first got %v", row["signal_id"])
	}
}

func TestPickCombinationStableWhenAPIOrderFlips(t *testing.T) {
	a := map[string]any{"name": "SAR MACD X", "signal_id": "id-a", "buy_signal": []any{map[string]any{"index": "SAR"}}}
	b := map[string]any{"name": "SAR MACD Y", "signal_id": "id-b", "buy_signal": []any{map[string]any{"index": "SAR"}}}
	row1, _ := PickCombination([]map[string]any{a, b}, "SAR MACD")
	row2, _ := PickCombination([]map[string]any{b, a}, "SAR MACD")
	if row1["signal_id"] != row2["signal_id"] {
		t.Fatalf("order flip changed pick: %v vs %v", row1["signal_id"], row2["signal_id"])
	}
}
