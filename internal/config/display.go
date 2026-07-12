package config

import "strings"

// Display detail modes aligned with Hermes details_mode.
const (
	ModeHidden    = "hidden"
	ModeCollapsed = "collapsed"
	ModeExpanded  = "expanded"
)

// DisplaySections are optional per-section overrides (empty = follow global).
type DisplaySections struct {
	Thinking string `json:"thinking,omitempty"`
	Tools    string `json:"tools,omitempty"`
	Activity string `json:"activity,omitempty"`
}

// DisplayConfig controls chat TUI density (Hermes display.* parity).
type DisplayConfig struct {
	Interface       string          `json:"interface,omitempty"` // tui | cli
	DetailsMode     string          `json:"details_mode,omitempty"`
	Sections        DisplaySections `json:"sections,omitempty"`
	MouseTracking   string          `json:"mouse_tracking,omitempty"`
	StatusIndicator string          `json:"status_indicator,omitempty"`
	ShowReasoning   *bool           `json:"show_reasoning,omitempty"`
}

// Normalize fills defaults and lowercases known enums.
func (d *DisplayConfig) Normalize() {
	if d == nil {
		return
	}
	d.Interface = strings.ToLower(strings.TrimSpace(d.Interface))
	if d.Interface == "" {
		d.Interface = "tui"
	}
	d.DetailsMode = normalizeMode(d.DetailsMode, ModeCollapsed)
	d.Sections.Thinking = normalizeModeAllowEmpty(d.Sections.Thinking)
	d.Sections.Tools = normalizeModeAllowEmpty(d.Sections.Tools)
	d.Sections.Activity = normalizeModeAllowEmpty(d.Sections.Activity)
	if d.Sections.Activity == "" {
		// Spec default: activity hidden
		d.Sections.Activity = ModeHidden
	}
	d.MouseTracking = strings.ToLower(strings.TrimSpace(d.MouseTracking))
	if d.MouseTracking == "" {
		d.MouseTracking = "wheel"
	}
	d.StatusIndicator = strings.ToLower(strings.TrimSpace(d.StatusIndicator))
	if d.StatusIndicator == "" {
		d.StatusIndicator = "emoji"
	}
}

func normalizeMode(s, fallback string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case ModeHidden, ModeCollapsed, ModeExpanded:
		return s
	default:
		return fallback
	}
}

func normalizeModeAllowEmpty(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", ModeHidden, ModeCollapsed, ModeExpanded:
		return s
	default:
		return ""
	}
}

// EffectiveMode returns the mode for a section name (thinking|tools|activity).
func (d DisplayConfig) EffectiveMode(section string) string {
	d.Normalize()
	var override string
	switch strings.ToLower(strings.TrimSpace(section)) {
	case "thinking":
		override = d.Sections.Thinking
	case "tools":
		override = d.Sections.Tools
	case "activity":
		override = d.Sections.Activity
	}
	if override != "" {
		return override
	}
	return d.DetailsMode
}

// ReasoningVisible reports whether thinking blocks should render.
func (d DisplayConfig) ReasoningVisible() bool {
	if d.ShowReasoning == nil {
		return true
	}
	return *d.ShowReasoning
}

// CycleDetailsMode advances hidden → collapsed → expanded → hidden.
func CycleDetailsMode(mode string) string {
	switch normalizeMode(mode, ModeCollapsed) {
	case ModeHidden:
		return ModeCollapsed
	case ModeCollapsed:
		return ModeExpanded
	default:
		return ModeHidden
	}
}
