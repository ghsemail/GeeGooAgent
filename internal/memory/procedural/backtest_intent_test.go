package procedural

import "testing"

func TestBacktestRunIntent(t *testing.T) {
	if !BacktestRunIntent("帮我回测一下sar信号加macd趋势在小米") {
		t.Fatal("expected backtest intent")
	}
	if BacktestRunIntent("用现成的来回测，不要新建") {
		t.Fatal("reuse phrasing alone should not count as backtest run intent")
	}
	if BacktestRunIntent("帮我测一下有没有买卖信号") {
		t.Fatal("signal-only probe should not count as backtest run intent")
	}
	if BacktestRunIntent("验证策略文档里的说法") {
		t.Fatal("generic strategy talk should not count as backtest run intent")
	}
	if BacktestRunIntent("帮我看看收益率高的基金") {
		t.Fatal("收益率 alone should not count as backtest run intent")
	}
}

func TestBacktestContinueIntent(t *testing.T) {
	if !BacktestContinueIntent("就用刚才那套") {
		t.Fatal("expected continue intent")
	}
	if ShouldBlockLegacyBacktestTools("帮我做 dca 定投回测") {
		t.Fatal("dca bypass should not block legacy tools")
	}
	if ShouldBlockLegacyBacktestTools("就用刚才那套") {
		t.Fatal("continue phrasing without backtest verb should not block legacy tools")
	}
	if !ShouldBlockLegacyBacktestTools("帮我回测一下") {
		t.Fatal("expected legacy tool block for explicit backtest")
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

func TestPrioritizeSignalProbePlaybook(t *testing.T) {
	loader := NewLoader("../../../skills")
	msg := "请帮我用「SAR抛物线」策略测试一下腾讯控股（0700.HK · 港股）。"
	if !SignalProbeIntent(msg) {
		t.Fatal("expected signal probe intent")
	}
	matched := loader.Match(msg, 2)
	got := loader.PrioritizeSignalProbePlaybook(msg, matched, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 prioritized matches, got %d: %v", len(got), skillNamesForTest(got))
	}
	if got[0].Name != "strategy-signal-probe" {
		t.Fatalf("first skill=%q want strategy-signal-probe", got[0].Name)
	}
	for _, sk := range got {
		if sk.Name == "strategy-backtest-run" {
			t.Fatal("strategy-backtest-run should be dropped for signal probe intent")
		}
	}
}

func TestShouldBlockSmartTradeBacktestTools(t *testing.T) {
	msg := "请帮我用「SAR抛物线」策略测试一下腾讯控股（0700.HK · 港股）。"
	if !ShouldBlockSmartTradeBacktestTools(msg) {
		t.Fatal("expected smart trade backtest block for signal probe")
	}
	if ShouldBlockSmartTradeBacktestTools("帮我回测一下 sar+macd 在小米") {
		t.Fatal("explicit backtest should not block run_strategy_backtest")
	}
}

func skillNamesForTest(skills []Skill) []string {
	out := make([]string, len(skills))
	for i, sk := range skills {
		out[i] = sk.Name
	}
	return out
}
