package playbookexec

import "testing"

func TestPickCustomSignalByName(t *testing.T) {
	items := []map[string]any{
		{"name": "4小时MACD市场节奏", "signal_id": "custom-a", "custom": map[string]any{"index": "Macd4HRhythm", "type": "signal"}},
		{"name": "MACD周期共振", "signal_id": "custom-b", "custom": map[string]any{"index": "MACDResonance", "type": "signal"}},
	}
	row, err := pickCustomSignal(items, "4小时MACD市场节奏")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "custom-a" {
		t.Fatalf("picked=%v", row["signal_id"])
	}
}

func TestCustomSignalRulesFromCustomBlock(t *testing.T) {
	buy, sell, err := customSignalRules(map[string]any{
		"custom": map[string]any{"index": "Macd4HRhythm", "type": "signal", "param": map[string]any{"x": 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(buy) != 1 || buy[0].(map[string]any)["index"] != "Macd4HRhythm" {
		t.Fatalf("buy=%v", buy)
	}
	if len(sell) != 1 {
		t.Fatalf("sell=%v", sell)
	}
}
