package config

// SkillsConfig controls procedural SKILL.md loading and extension scripts.
type SkillsConfig struct {
	Disabled   []string                        `json:"disabled,omitempty"`
	Extensions map[string]ExtensionSkillConfig `json:"extensions,omitempty"`
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
