package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestResolveInboundSkillOptionsLegacyOnly(t *testing.T) {
	legacy := map[string]any{
		"signal_diagnose": map[string]any{"frequency": "60m"},
	}
	got := ResolveInboundSkillOptions(nil, legacy)
	block, _ := got["signal_diagnose"].(map[string]any)
	if block["frequency"] != "60m" {
		t.Fatalf("legacy frequency: %+v", got)
	}
}

func TestResolveInboundSkillOptionsSkillOnly(t *testing.T) {
	skill := map[string]any{
		"locked_skill":    "signal_diagnose",
		"signal_diagnose": map[string]any{"frequency": "60m", "lab_mode": true},
	}
	got := ResolveInboundSkillOptions(skill, nil)
	if ParseLockedSkill(got) != "signal_diagnose" {
		t.Fatalf("locked: %+v", got)
	}
	o := ParseSignalDiagnoseOptsFromSkillOptions(got)
	if o == nil || o.Frequency != "60m" || o.LabMode == nil || !*o.LabMode {
		t.Fatalf("opts: %+v", o)
	}
}

func TestResolveInboundSkillOptionsSkillWinsPerKey(t *testing.T) {
	legacy := map[string]any{
		"signal_diagnose": map[string]any{"frequency": "15m"},
		"locked_skill":    "other",
	}
	skill := map[string]any{
		"signal_diagnose": map[string]any{"frequency": "60m"},
		"locked_skill":    "signal_diagnose",
	}
	got := ResolveInboundSkillOptions(skill, legacy)
	if ParseLockedSkill(got) != "signal_diagnose" {
		t.Fatalf("locked_skill should come from skill_options: %+v", got)
	}
	block, _ := got["signal_diagnose"].(map[string]any)
	if block["frequency"] != "60m" {
		t.Fatalf("frequency should come from skill_options: %+v", got)
	}
}

func TestApplyInboundSkillOptionsFrequencyClarifySkip(t *testing.T) {
	session := &runtime.Session{}
	ApplyInboundSkillOptions(session, map[string]any{
		"locked_skill": "signal_diagnose",
		"signal_diagnose": map[string]any{
			"frequency": "60m",
		},
	}, nil)
	if session.LockedSkill != "signal_diagnose" {
		t.Fatalf("locked=%q", session.LockedSkill)
	}
	ans, ok := runtime.TryAnswerFrequencyFromPanelOps(session, "请选择 K 线周期", []string{"60m", "daily"})
	if !ok || ans != "60m" {
		t.Fatalf("expected auto frequency clarify answer, got %q ok=%v", ans, ok)
	}
}
