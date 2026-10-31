package chat

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// ResolveInboundSkillOptions merges Monday chat skill_options with legacy workflow_options.
// Per-key values from skill_options win when both maps are present.
func ResolveInboundSkillOptions(skillOptions, legacyWorkflow map[string]any) map[string]any {
	if len(skillOptions) == 0 && len(legacyWorkflow) == 0 {
		return nil
	}
	out := make(map[string]any)
	for k, v := range legacyWorkflow {
		out[k] = v
	}
	for k, v := range skillOptions {
		out[k] = v
	}
	return out
}

// ParseLockedSkill reads skill_options.locked_skill (or legacy workflow_options bag).
func ParseLockedSkill(raw map[string]any) string {
	if len(raw) == 0 {
		return ""
	}
	s, _ := raw["locked_skill"].(string)
	return strings.TrimSpace(s)
}

// ApplyInboundSkillOptions maps resolved skill_options onto the live runtime session for one turn.
func ApplyInboundSkillOptions(session *runtime.Session, skillOptions, legacyWorkflow map[string]any) {
	if session == nil {
		return
	}
	resolved := ResolveInboundSkillOptions(skillOptions, legacyWorkflow)
	if len(resolved) == 0 {
		return
	}
	session.LockedSkill = ParseLockedSkill(resolved)
	if opts := ParseSignalDiagnoseOptsFromSkillOptions(resolved); opts != nil {
		session.InboundPanelOps = opts
	}
	if opts := ParseStrategyDevOptsFromSkillOptions(resolved); opts != nil {
		session.PendingStrategyDevOpts = opts
	}
}
