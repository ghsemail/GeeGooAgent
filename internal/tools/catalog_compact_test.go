package tools

import "testing"

func TestCompactSignalCombinations(t *testing.T) {
	payload := []any{
		map[string]any{
			"signal_id": "abc123",
			"name":      "MACD共振",
			"brief":     "短简介",
			"info":      stringsRepeat("长说明", 200),
			"frequency": "60m",
			"buy_signal": []any{
				map[string]any{"index": "MACD", "type": "signal"},
				map[string]any{"index": "RSI", "type": "signal"},
			},
			"sell_signal": []any{
				map[string]any{"index": "MACD", "type": "signal"},
			},
		},
	}
	compact := compactHTTPPayload("get_signal_combinations", payload)
	items, ok := compact.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected compact: %#v", compact)
	}
	row := items[0].(map[string]any)
	if row["info"] != nil {
		t.Fatal("info should be stripped")
	}
	if row["buy_signal"] != nil {
		t.Fatal("buy_signal should be stripped")
	}
	if row["buy_rule_count"] != 2 {
		t.Fatalf("buy_rule_count=%v", row["buy_rule_count"])
	}
	if row["buy_indexes"] != "MACD+RSI" {
		t.Fatalf("buy_indexes=%v", row["buy_indexes"])
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
