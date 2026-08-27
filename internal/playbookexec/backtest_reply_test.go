package playbookexec

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSignalIDFromRow(t *testing.T) {
	row := map[string]any{"signal_id": "662d0424c4cee7ffb800d0af", "name": "SAR+MACD"}
	if got := signalIDFromRow(row); got != "662d0424c4cee7ffb800d0af" {
		t.Fatalf("signal_id=%q", got)
	}
	if signalIDFromRow(nil) != "" {
		t.Fatal("nil row should return empty")
	}
}

func TestBuildBacktestLLMPayloadIncludesTradesWithoutChartData(t *testing.T) {
	trades := make([]any, 0, 17)
	for i := 0; i < 17; i++ {
		trades = append(trades, map[string]any{
			"time": "2026-05-13 14:00:00", "action": "signalBuy", "action_label": "信号买入",
			"trade_price": 463.4, "position": 100,
		})
	}
	logDetail := map[string]any{
		"trades": trades,
		"run": map[string]any{
			"strategy_ids": []any{"662d0424c4cee7ffb800d0af"},
			"chart_data": map[string]any{
				"probe": map[string]any{"bars": make([]any, 462)},
			},
			"probe_snapshot": map[string]any{
				"buy_rules": []any{map[string]any{"index": "SAR"}},
			},
		},
	}
	payload := buildBacktestLLMPayload(backtestReplyInput{
		Code: "00700.HK", Name: "腾讯控股", SignalID: "662d0424c4cee7ffb800d0af",
		LogDetail: logDetail, RunSummary: map[string]any{"log_id": "x", "profit_rate": 1.2},
	})
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 20000 {
		t.Fatalf("payload too large: %d bytes", len(raw))
	}
	tr, ok := payload["trades"].([]map[string]any)
	if !ok || len(tr) != 17 {
		t.Fatalf("trades=%v", payload["trades"])
	}
	if tr[0]["price"] == nil || tr[0]["action"] == nil {
		t.Fatalf("normalized trade=%v", tr[0])
	}
	if _, ok := payload["stock"]; !ok {
		t.Fatalf("payload should use human sections: %v", payload)
	}
	if strings.Contains(string(raw), "signal_id") || strings.Contains(string(raw), "months_back") {
		t.Fatal("payload should not contain internal field names")
	}
}

func TestFormatTradesTableUsesSignalAPIFields(t *testing.T) {
	table := formatTradesTable(map[string]any{
		"trades": []any{
			map[string]any{
				"time": "2026-05-13 14:00:00", "action": "signalBuy", "action_label": "信号买入",
				"trade_price": 463.4, "position": 100,
			},
		},
	})
	if !stringsContains(table, "信号买入") || !stringsContains(table, "463.4") {
		t.Fatalf("table=%q", table)
	}
}

func TestStripReasoningTags(t *testing.T) {
	in := "<think>hidden</think>\n\n## 回测结果"
	if got := stripReasoningTags(in); got != "## 回测结果" {
		t.Fatalf("got=%q", got)
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
			"log_id":      "log-123",
			"profit_rate": 12.5,
			"final_value": 112500,
			"drawdown":    4.2,
			"trade_count": 2,
		},
		LogDetail: map[string]any{
			"trades": []any{
				map[string]any{"time": "2026-01-01", "action_label": "信号买入", "trade_price": 300, "position": 100},
				map[string]any{"time": "2026-02-01", "action_label": "止损", "trade_price": 320, "position": 100},
			},
		},
	})
	if !stringsContains(reply, "策略回测") {
		t.Fatalf("missing backtest title: %s", reply)
	}
	if !stringsContains(reply, "成交记录") || !stringsContains(reply, "信号买入") {
		t.Fatalf("missing trades section: %s", reply)
	}
	if stringsContains(reply, "log_id") || stringsContains(reply, "signal_id") {
		t.Fatalf("should not expose internal ids: %s", reply)
	}
}

func stringsContains(s, sub string) bool {
	return strings.Contains(s, sub)
}
