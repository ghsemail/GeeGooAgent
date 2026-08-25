package tools

import "testing"

func TestCompactProbeSeries(t *testing.T) {
	payload := map[string]any{
		"code":        "AAPL.US",
		"frequency":   "60m",
		"months_back": 1,
		"bars": []any{
			map[string]any{"time": "2026-08-01 10:00:00"},
			map[string]any{"time": "2026-08-02 11:00:00"},
			map[string]any{"time": "2026-08-03 12:00:00"},
		},
		"buy_merged":  []any{0, 1, 0},
		"sell_merged": []any{0, 0, -1},
	}
	compact := compactHTTPPayload(Context{}, "probe_bot_signal_series", payload)
	row, ok := compact.(map[string]any)
	if !ok {
		t.Fatalf("unexpected compact: %#v", compact)
	}
	if row["bars"] != nil {
		t.Fatal("bars should be stripped")
	}
	if row["buy_hits"] != 1 {
		t.Fatalf("buy_hits=%v", row["buy_hits"])
	}
	if row["sell_hits"] != 1 {
		t.Fatalf("sell_hits=%v", row["sell_hits"])
	}
	recent, ok := row["recent_buy_times"].([]string)
	if !ok || len(recent) != 1 || recent[0] != "2026-08-02 11:00:00" {
		t.Fatalf("recent_buy_times=%v", row["recent_buy_times"])
	}
}

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
	compact := compactHTTPPayload(Context{}, "get_signal_combinations", payload)
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
