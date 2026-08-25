package playbookexec

import "testing"

func TestRouteBacktestRun(t *testing.T) {
	skills := []string{"strategy-backtest-run", "strategy-backtest"}
	if p, ok := Route(skills, "帮我回测一下 sar 信号加 macd 在小米"); !ok || p != playbookBacktestRun {
		t.Fatalf("route=%q ok=%v", p, ok)
	}
	if _, ok := Route(skills, "帮我做 dca 定投回测"); ok {
		t.Fatal("dca bypass should not route")
	}
	if _, ok := Route([]string{"strategy-signal-probe"}, "帮我回测一下"); ok {
		t.Fatal("missing backtest playbook should not route")
	}
}

func TestHeuristicBacktestPlan(t *testing.T) {
	plan := heuristicBacktestPlan("帮我测试一下帮我回测一下sar信号加macd趋势在小米")
	if plan.StockQuery != "小米" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if plan.SignalKind != "combination" {
		t.Fatalf("kind=%q", plan.SignalKind)
	}
}

func TestPickCombination(t *testing.T) {
	items := []map[string]any{
		{"name": "SAR信号配套MACD直方图趋势", "signal_id": "a", "buy_signal": []any{map[string]any{"index": "SAR"}}},
		{"name": "RSI阈值信号", "signal_id": "b", "buy_signal": []any{map[string]any{"index": "RSI"}}},
	}
	row, err := pickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "a" {
		t.Fatalf("picked=%v", row["signal_id"])
	}
}
