package slots

import "testing"

func TestShouldResolveNewSymbolForProbeBareHKCode(t *testing.T) {
	if !ShouldResolveNewSymbolForProbe("00700") {
		t.Fatal("bare 00700 should trigger symbol resolve")
	}
}

func TestShouldResolveNewSymbolForProbeStrategySwapOnly(t *testing.T) {
	if ShouldResolveNewSymbolForProbe("换一个策略") {
		t.Fatal("strategy-only swap must not trigger symbol resolve")
	}
}

func TestShouldResolveNewSymbolForProbeSymbolSwapPhrase(t *testing.T) {
	if !ShouldResolveNewSymbolForProbe("换一个标的试试") {
		t.Fatal("symbol swap phrase should trigger symbol resolve")
	}
}

func TestShouldResolveNewSymbolForProbeClarifyAnswer(t *testing.T) {
	if !ShouldResolveNewSymbolForProbe("腾讯控股（00700.HK）") {
		t.Fatal("clarify answer with HK code should trigger symbol resolve")
	}
}
