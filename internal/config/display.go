package config

import "strings"

// Display detail modes aligned with Hermes details_mode.
const (
	ModeHidden    = "hidden"
	ModeCollapsed = "collapsed"
	ModeExpanded  = "expanded"
)

// Reply format for assistant messages in the TUI/CLI.
const (
	ReplyFormatMarkdown = "markdown"
	ReplyFormatPlain    = "plain"
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
	MouseTracking   string          `json:"mouse_tracking,omitempty"` // off|wheel|buttons|all — off allows mouse text selection
	AltScreen       *bool           `json:"alt_screen,omitempty"`     // default true; false uses main buffer (scrollback + select)
	StatusIndicator string          `json:"status_indicator,omitempty"`
	ShowReasoning   *bool           `json:"show_reasoning,omitempty"`
	StreamReply     *bool           `json:"stream_reply,omitempty"`  // default false: wait for full reply then render
	ReplyFormat     string          `json:"reply_format,omitempty"` // markdown | plain (default markdown)
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
		d.MouseTracking = "off"
	}
	d.StatusIndicator = strings.ToLower(strings.TrimSpace(d.StatusIndicator))
	if d.StatusIndicator == "" {
		d.StatusIndicator = "emoji"
	}
	d.ReplyFormat = strings.ToLower(strings.TrimSpace(d.ReplyFormat))
	switch d.ReplyFormat {
	case "", ReplyFormatMarkdown:
		d.ReplyFormat = ReplyFormatMarkdown
	case ReplyFormatPlain:
	default:
		d.ReplyFormat = ReplyFormatMarkdown
	}
}

// StreamReplyEnabled reports whether assistant text streams token-by-token in the UI.
func (d DisplayConfig) StreamReplyEnabled() bool {
	d.Normalize()
	return d.StreamReply != nil && *d.StreamReply
}

// ReplyMarkdownEnabled reports whether completed replies use glamour markdown rendering.
func (d DisplayConfig) ReplyMarkdownEnabled() bool {
	d.Normalize()
	return d.ReplyFormat == ReplyFormatMarkdown
}

// AltScreenEnabled reports whether the TUI uses the terminal alternate screen buffer.
func (d DisplayConfig) AltScreenEnabled() bool {
	d.Normalize()
	if d.AltScreen == nil {
		return true
	}
	return *d.AltScreen
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
