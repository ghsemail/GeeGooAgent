package runtime

import (
	"strings"
	"testing"
)

func TestPanelOpsTurnHintInboundFrequency(t *testing.T) {
	session := &Session{
		InboundPanelOps: &SignalDiagnoseOpts{
			Frequency:          "60m",
			MonthsBack:         3,
			PanelStrategyLabel: "买:SAR · 卖:SAR",
		},
	}
	hint := PanelOpsTurnHint(session)
	if hint == "" {
		t.Fatal("expected hint")
	}
	if !strings.Contains(hint, "60m") || !strings.Contains(strings.ToLower(hint), "clarify") {
		t.Fatalf("hint=%q", hint)
	}
}

func TestPanelOpsTurnHintStickyOnly(t *testing.T) {
	session := &Session{
		StickyPanelOps: &SignalDiagnoseOpts{Frequency: "15m"},
	}
	hint := PanelOpsTurnHint(session)
	if hint == "" || !strings.Contains(hint, "15m") {
		t.Fatalf("hint=%q", hint)
	}
}

func TestPanelOpsTurnHintEmptyWithoutFrequency(t *testing.T) {
	session := &Session{
		InboundPanelOps: &SignalDiagnoseOpts{MonthsBack: 3},
	}
	if PanelOpsTurnHint(session) == "" {
		t.Fatal("expected hint for months_back")
	}
}

func TestTryAnswerFrequencyFromPanelOpsInbound(t *testing.T) {
	session := &Session{
		InboundPanelOps: &SignalDiagnoseOpts{Frequency: "60m"},
	}
	ans, ok := TryAnswerFrequencyFromPanelOps(session, "请选择 K 线周期", []string{"60m", "daily", "5m", "15m"})
	if !ok || ans != "60m" {
		t.Fatalf("ok=%v ans=%q", ok, ans)
	}
}

func TestTryAnswerFrequencyFromPanelOpsSticky(t *testing.T) {
	session := &Session{
		StickyPanelOps: &SignalDiagnoseOpts{Frequency: "15m"},
	}
	ans, ok := TryAnswerFrequencyFromPanelOps(session, "请选择 K 线周期", []string{"60m", "daily"})
	if !ok || ans != "15m" {
		t.Fatalf("ok=%v ans=%q", ok, ans)
	}
}

func TestTryAnswerFrequencyFromPanelOpsNoPanelAllowsClarify(t *testing.T) {
	session := &Session{}
	if _, ok := TryAnswerFrequencyFromPanelOps(session, "请选择 K 线周期", []string{"60m", "daily"}); ok {
		t.Fatal("expected no bypass without panel frequency")
	}
}

func TestTryAnswerFrequencyFromPanelOpsIgnoresNonFrequencyClarify(t *testing.T) {
	session := &Session{
		InboundPanelOps: &SignalDiagnoseOpts{Frequency: "60m"},
	}
	if _, ok := TryAnswerFrequencyFromPanelOps(session, "请选择策略", []string{"SAR", "MACD"}); ok {
		t.Fatal("should not bypass strategy clarify")
	}
}
