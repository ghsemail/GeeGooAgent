package notify

import "testing"

func TestFormatTradeMarkdown_signal(t *testing.T) {
	text := FormatTradeMarkdown(2, map[string]any{
		"trade_env":   "REAL",
		"user":        "ghsemail",
		"botname":     "测试",
		"botType":     "SmartReminder",
		"code":        "600519.SH",
		"stock_name":  "贵州茅台",
		"buy_signal":  1,
		"sell_signal": -1,
		"next_opt":    "buy",
		"trade_agent": map[string]any{"result": "buy", "confidence": "high"},
	})
	if text == "" {
		t.Fatal("empty markdown")
	}
	if !contains(text, "智能体决策") || !contains(text, "high") {
		t.Fatalf("missing agent block: %q", text)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
