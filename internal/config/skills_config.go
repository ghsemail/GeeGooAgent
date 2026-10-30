package config

import "strings"

// SkillsConfig controls procedural SKILL.md loading and extension scripts.
type SkillsConfig struct {
	Disabled   []string                        `json:"disabled,omitempty"`
	Extensions map[string]ExtensionSkillConfig `json:"extensions,omitempty"`
	// SkillInjection: keyword (legacy Match full inject) | catalog (default) | off.
	SkillInjection string `json:"skill_injection,omitempty"`
	// ChatOpenFlowTool registers run_chat_workflow_turn on the Chat allowlist (deprecated; default false).
	ChatOpenFlowTool *bool `json:"chat_open_flow_tool,omitempty"`
}

// ExtensionSkillConfig toggles an optional extension skill (scripts under skills/extensions/).
type ExtensionSkillConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// EffectiveSkills returns skills settings from config.
func (c *AppConfig) EffectiveSkills() SkillsConfig {
	if c == nil {
		return SkillsConfig{}
	}
	return c.Skills
}

// EffectiveSkillInjection returns chat skill injection mode (default catalog).
func (c *AppConfig) EffectiveSkillInjection() string {
	if c == nil {
		return "catalog"
	}
	if s := strings.TrimSpace(c.Skills.SkillInjection); s != "" {
		return strings.ToLower(s)
	}
	return "catalog"
}

// EffectiveChatOpenFlowTool is true when the legacy run_chat_workflow_turn tool stays on the Chat allowlist.
func (c *AppConfig) EffectiveChatOpenFlowTool() bool {
	if c == nil || c.Skills.ChatOpenFlowTool == nil {
		return false
	}
	return *c.Skills.ChatOpenFlowTool
}
