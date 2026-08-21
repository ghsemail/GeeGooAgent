package config

// ContextProfilesConfig bounds AGENTS.md loading for Chat.
type ContextProfilesConfig struct {
	MaxMergedBytes        int  `json:"max_merged_bytes,omitempty"`
	MaxProfilesPerSession int  `json:"max_profiles_per_session,omitempty"`
	LoadCursorRules       bool `json:"load_cursor_rules,omitempty"`
}

const (
	defaultProfileMaxMergedBytes        = 32768
	defaultProfileMaxProfilesPerSession = 4
)

// EffectiveContextProfileMaxMergedBytes returns merged AGENTS byte cap.
func (c *AppConfig) EffectiveContextProfileMaxMergedBytes() int {
	if c != nil && c.ContextProfiles.MaxMergedBytes > 0 {
		return c.ContextProfiles.MaxMergedBytes
	}
	return defaultProfileMaxMergedBytes
}

// EffectiveContextProfileMaxPerSession returns max session profile refs.
func (c *AppConfig) EffectiveContextProfileMaxPerSession() int {
	if c != nil && c.ContextProfiles.MaxProfilesPerSession > 0 {
		return c.ContextProfiles.MaxProfilesPerSession
	}
	return defaultProfileMaxProfilesPerSession
}
