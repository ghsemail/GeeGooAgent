package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestParseSignalDiagnoseOptsLabMode(t *testing.T) {
	on := true
	o := ParseSignalDiagnoseOptsFromWorkflowOptions(map[string]any{
		"signal_diagnose": map[string]any{"lab_mode": true},
	})
	if o == nil || o.LabMode == nil || !*o.LabMode {
		t.Fatal("expected lab_mode true")
	}
	_ = on
}

func TestCardPayloadLabModeSummaries(t *testing.T) {
	flow := &Flow{
		Template: SkillSignalDiagnose,
		LabMode:  true,
		LabEpisodeSummaries: strategydev.LabCardEpisodeSummaries{
			"buy:0": {DetailRef: "buy:0", KlinePattern: "hammer", PriceAnalysis: "综合分 1.2"},
		},
		LabFlowPriceSummary: "综合分 1.2",
		SignalEval:          slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	out := CardPayload(flow)
	if out["lab_mode"] != true {
		t.Fatal("expected lab_mode in card")
	}
	if out["lab_episode_summaries"] == nil {
		t.Fatal("expected lab_episode_summaries")
	}
}

func TestApplyInboundPanelOpsToFlowLabMode(t *testing.T) {
	on := true
	session := &runtime.Session{InboundPanelOps: &runtime.SignalDiagnoseOpts{LabMode: &on}}
	flow := newSignalDiagnoseFlow("诊断 x · y")
	ApplyInboundPanelOpsToFlow(session, flow)
	if !flow.LabMode {
		t.Fatal("expected lab mode on flow")
	}
}
