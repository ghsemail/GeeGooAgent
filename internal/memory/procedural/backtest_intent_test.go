package procedural

import "testing"

func TestBacktestRunIntent(t *testing.T) {
	if !BacktestRunIntent("帮我回测一下sar信号加macd趋势在小米") {
		t.Fatal("expected backtest intent")
	}
	if !BacktestRunIntent("用现成的来回测，不要新建") {
		t.Fatal("expected backtest intent for reuse message")
	}
	if BacktestRunIntent("帮我测一下有没有买卖信号") {
		t.Fatal("signal-only probe should not count as backtest run intent")
	}
}

func TestBacktestContinueIntent(t *testing.T) {
	if !BacktestContinueIntent("就用刚才那套") {
		t.Fatal("expected continue intent")
	}
	if BacktestContinueIntent("刚才那个收益是多少") {
		t.Fatal("asking about last result is not continue-backtest intent")
	}
	if BacktestContinueIntent("同样看看行情") {
		t.Fatal("generic 同样/刚才 should not count as continue intent")
	}
	if ShouldBlockLegacyBacktestTools("帮我做 dca 定投回测") {
		t.Fatal("dca bypass should not block legacy tools")
	}
	if !ShouldBlockLegacyBacktestTools("用现成的来回测") {
		t.Fatal("expected legacy tool block")
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
