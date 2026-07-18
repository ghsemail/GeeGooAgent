package chattui

import (
	"strings"
	"unicode/utf8"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// SectionKind identifies a collapsible transcript section.
type SectionKind string

const (
	KindThinking SectionKind = "thinking"
	KindTools    SectionKind = "tools"
	KindActivity SectionKind = "activity"
	KindReply    SectionKind = "reply"
	KindUser     SectionKind = "user"
)

const thinkingPreviewLines = 2

// IsProcessKind reports thinking, tools, or activity blocks.
func IsProcessKind(kind SectionKind) bool {
	return kind == KindThinking || kind == KindTools || kind == KindActivity
}

// Block is one collapsible (or always-open) transcript unit.
type Block struct {
	ID           string
	Kind         SectionKind
	Title        string
	Body         string
	Live         bool
	UserExpanded *bool // nil = follow EffectiveMode
	LineHint     int
	DurationSec  float64
}

// IsExpanded reports whether the full body should be visible.
func (b Block) IsExpanded(cfg config.DisplayConfig) bool {
	if b.Kind == KindReply || b.Kind == KindUser {
		return true
	}
	if b.Kind == KindThinking && !cfg.ReasoningVisible() {
		return false
	}
	if b.Live {
		if b.UserExpanded != nil {
			return *b.UserExpanded
		}
		return cfg.EffectiveMode(string(b.Kind)) == config.ModeExpanded
	}
	if b.UserExpanded != nil {
		return *b.UserExpanded
	}
	mode := cfg.EffectiveMode(string(b.Kind))
	switch mode {
	case config.ModeExpanded:
		return true
	case config.ModeHidden, config.ModeCollapsed:
		return false
	default:
		return false
	}
}

// ShowLivePreview reports whether a single current line should show while Live (tools).
func (b Block) ShowLivePreview(cfg config.DisplayConfig) bool {
	if !b.Live || b.Kind == KindReply || b.Kind == KindUser || b.Kind == KindThinking {
		return false
	}
	if b.IsExpanded(cfg) {
		return false
	}
	return strings.TrimSpace(b.Body) != ""
}

// ShowThinkingPreview reports whether up to two thinking lines should show when collapsed.
func (b Block) ShowThinkingPreview(cfg config.DisplayConfig) bool {
	if b.Kind != KindThinking || !cfg.ReasoningVisible() {
		return false
	}
	if b.IsExpanded(cfg) {
		return false
	}
	return strings.TrimSpace(b.Body) != ""
}

// LastBodyLines returns up to n trailing non-empty lines from Body.
func (b Block) LastBodyLines(n int) []string {
	body := strings.TrimRight(b.Body, "\n")
	if body == "" || n <= 0 {
		return nil
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// LastBodyLine returns the most recent non-empty line in Body.
func (b Block) LastBodyLine() string {
	body := strings.TrimRight(b.Body, "\n")
	if body == "" {
		return ""
	}
	if idx := strings.LastIndex(body, "\n"); idx >= 0 {
		return body[idx+1:]
	}
	return body
}

// IsVisible reports whether the block header (and maybe body) should render at all.
func (b Block) IsVisible(cfg config.DisplayConfig) bool {
	if b.Kind == KindReply || b.Kind == KindUser {
		return true
	}
	if b.Kind == KindThinking && !cfg.ReasoningVisible() {
		return false
	}
	if b.Live {
		return true
	}
	if b.UserExpanded != nil && *b.UserExpanded {
		return true
	}
	return cfg.EffectiveMode(string(b.Kind)) != config.ModeHidden
}

// Chevron returns ▾ when expanded, ▸ when collapsed.
func (b Block) Chevron(cfg config.DisplayConfig) string {
	if b.IsExpanded(cfg) {
		return "▾"
	}
	return "▸"
}

// LineCount estimates body lines for summary headers.
func (b Block) LineCount() int {
	if b.LineHint > 0 {
		return b.LineHint
	}
	body := strings.TrimSpace(b.Body)
	if body == "" {
		return 0
	}
	return strings.Count(body, "\n") + 1
}

// ToggleExpand flips user override (nil → expand if currently collapsed).
func (b *Block) ToggleExpand(cfg config.DisplayConfig) {
	cur := b.IsExpanded(cfg)
	next := !cur
	b.UserExpanded = &next
}

// TruncateRunes shortens s to at most n runes with ellipsis.
func TruncateRunes(s string, n int) string {
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
