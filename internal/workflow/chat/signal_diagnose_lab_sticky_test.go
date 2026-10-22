package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestLabTogglePreservesPanelBuySellOverrides(t *testing.T) {
	buy := []any{map[string]any{"index": "SAR", "type": "signal"}}
	sell := []any{map[string]any{"index": "MACD", "type": "flag"}}
	labOff := false
	labOn := true

	session := &runtime.Session{}
	session.PendingSignalDiagnoseOpts = &runtime.SignalDiagnoseOpts{
		BuySignal: buy, SellSignal: sell, Frequency: "60m", MonthsBack: 3,
		PanelStrategyLabel: "买:SAR · 卖:MACD", LabMode: &labOff,
	}
	flowA := &Flow{Template: SkillSignalDiagnose}
	ApplyPendingSignalDiagnoseOpts(session, flowA)

	session.PendingSignalDiagnoseOpts = &runtime.SignalDiagnoseOpts{LabMode: &labOn}
	flowB := &Flow{Template: SkillSignalDiagnose}
	ApplyPendingSignalDiagnoseOpts(session, flowB)

	if len(flowB.ProbeBuyOverride) != 1 || len(flowB.ProbeSellOverride) != 1 {
		t.Fatalf("lab toggle dropped overrides: buy=%d sell=%d", len(flowB.ProbeBuyOverride), len(flowB.ProbeSellOverride))
	}
	if !flowB.LabMode || flowA.LabMode {
		t.Fatalf("lab flags A=%v B=%v", flowA.LabMode, flowB.LabMode)
	}
	fpA := fingerprintProbeSignalPlan(flowA)
	fpB := fingerprintProbeSignalPlan(flowB)
	if fpA == "" || fpA != fpB {
		t.Fatalf("signal plan fingerprint mismatch: %q vs %q", fpA, fpB)
	}
}

func TestCardProbeSignalPlanFingerprintStable(t *testing.T) {
	flow := &Flow{
		Template:               SkillSignalDiagnose,
		ProbeBuyOverride:       []any{map[string]any{"index": "SAR"}},
		ProbeSellOverride:      []any{map[string]any{"index": "SAR"}},
		ProbeFrequencyOverride: "60m",
		DiagnoseSignalSource:   "panel",
	}
	flow.ProbeSignalFingerprint = fingerprintProbeSignalPlan(flow)
	out := CardPayload(flow)
	plan, ok := out["probe_signal_plan"].(map[string]any)
	if !ok || plan["fingerprint"] != flow.ProbeSignalFingerprint {
		t.Fatalf("card plan=%v", out["probe_signal_plan"])
	}
}
