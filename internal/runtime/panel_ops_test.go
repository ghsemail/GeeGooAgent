package runtime

import "testing"

func TestApplyPanelOpsToSignalProbeToolOverridesLLM(t *testing.T) {
	session := &Session{
		InboundPanelOps: &SignalDiagnoseOpts{Frequency: "60m", MonthsBack: 6},
	}
	body := map[string]any{"frequency": "daily", "months_back": 3}
	ApplyPanelOpsToSignalProbeTool(session, body)
	if body["frequency"] != "60m" {
		t.Fatalf("frequency=%v", body["frequency"])
	}
	if body["months_back"] != 6 {
		t.Fatalf("months_back=%v", body["months_back"])
	}
	if session.InboundPanelOps != nil {
		t.Fatal("inbound should be consumed")
	}
	if session.StickyPanelOps == nil || session.StickyPanelOps.Frequency != "60m" {
		t.Fatalf("sticky=%+v", session.StickyPanelOps)
	}
}

func TestApplyPanelOpsStickyWithoutInbound(t *testing.T) {
	session := &Session{
		StickyPanelOps: &SignalDiagnoseOpts{Frequency: "15m"},
	}
	body := map[string]any{"frequency": "daily"}
	ApplyPanelOpsToSignalProbeTool(session, body)
	if body["frequency"] != "15m" {
		t.Fatalf("frequency=%v", body["frequency"])
	}
}

func TestApplyPanelOpsNoPanelLeavesBody(t *testing.T) {
	session := &Session{}
	body := map[string]any{"frequency": "daily"}
	ApplyPanelOpsToSignalProbeTool(session, body)
	if body["frequency"] != "daily" {
		t.Fatalf("frequency=%v", body["frequency"])
	}
}
