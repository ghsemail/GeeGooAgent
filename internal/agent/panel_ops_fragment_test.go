package agent

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestSignalDiagnosePanelOpsFragmentPresent(t *testing.T) {
	session := &runtime.Session{
		InboundPanelOps: &runtime.SignalDiagnoseOpts{Frequency: "60m"},
	}
	text := signalDiagnosePanelOpsFragment(session).Render()
	if !strings.Contains(text, "60m") {
		t.Fatalf("fragment=%q", text)
	}
}

func TestSignalDiagnosePanelOpsFragmentAbsent(t *testing.T) {
	if signalDiagnosePanelOpsFragment(&runtime.Session{}).Render() != "" {
		t.Fatal("expected empty fragment without panel ops")
	}
}
