package playbookexec

import "testing"

func TestSignalIDFromRow(t *testing.T) {
	row := map[string]any{"signal_id": "662d0424c4cee7ffb800d0af", "name": "SAR+MACD"}
	if got := signalIDFromRow(row); got != "662d0424c4cee7ffb800d0af" {
		t.Fatalf("signal_id=%q", got)
	}
	if signalIDFromRow(nil) != "" {
		t.Fatal("nil row should return empty")
	}
}

func TestFormatBacktestReplyFallbackIncludesConfigAndTrades(t *testing.T) {
	reply := formatBacktestReplyFallback(backtestReplyInput{
		Code:          "00700.HK",
		Name:          "腾讯控股",
		StrategyLabel: "SAR信号配套MACD直方图趋势",
		Frequency:     "60m",
		Period:        "3m",
		MonthsBack:    3,
		Fund:          100000,
		BaseOrderSize: 100,
		SignalID:      "662d0424c4cee7ffb800d0af",
		BuySignal:     []any{map[string]any{"index": "SAR", "type": "signal"}},
		SellSignal:    []any{map[string]any{"index": "SAR", "type": "signal"}},
		TradeConfig:   defaultSmartTradeTradeConfig(),
		RunSummary: map[string]any{
			"log_id":       "log-123",
			"profit_rate":  12.5,
			"final_value":  112500,
			"drawdown":     4.2,
			"trade_count":  2,
		},
		LogDetail: map[string]any{
			"trades": []any{
				map[string]any{"time": "2026-01-01", "side": "buy", "price": 300, "qty": 100},
				map[string]any{"time": "2026-02-01", "side": "sell", "price": 320, "qty": 100},
			},
		},
	})
	if !stringsContains(reply, "662d0424c4cee7ffb800d0af") {
		t.Fatalf("missing signal_id in reply: %s", reply)
	}
	if !stringsContains(reply, "成交明细") {
		t.Fatalf("missing trades table: %s", reply)
	}
	if !stringsContains(reply, "交易参数") {
		t.Fatalf("missing trade_config: %s", reply)
	}
}

func stringsContains(s, sub string) bool {
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
