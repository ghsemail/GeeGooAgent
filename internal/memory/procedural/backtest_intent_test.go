package procedural

import "testing"

func TestBacktestRunIntent(t *testing.T) {
	if !BacktestRunIntent("帮我回测一下sar信号加macd趋势在小米") {
		t.Fatal("expected backtest intent")
	}
	if BacktestRunIntent("帮我测一下有没有买卖信号") {
		t.Fatal("signal-only probe should not count as backtest run intent")
	}
}

func TestPrioritizeBacktestRunPlaybook(t *testing.T) {
	loader := NewLoader("../../../skills")
	msg := "帮我测试一下帮我回测一下sar信号加macd趋势在小米"
	matched := loader.Match(msg, 2)
	if len(matched) != 2 {
		t.Fatalf("expected 2 baseline matches, got %d", len(matched))
	}
	got := loader.PrioritizeBacktestRunPlaybook(msg, matched, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 prioritized matches, got %d", len(got))
	}
	if got[0].Name != "strategy-backtest-run" {
		t.Fatalf("first skill=%q want strategy-backtest-run", got[0].Name)
	}
	for _, sk := range got {
		if sk.Name == "strategy-signal-probe" {
			t.Fatal("strategy-signal-probe should be dropped for backtest intent at maxSkills=2")
		}
	}
}
