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

// IsExpanded reports whether body should be visible.
func (b Block) IsExpanded(cfg config.DisplayConfig) bool {
	if b.Kind == KindReply || b.Kind == KindUser {
		return true
	}
	if b.Kind == KindThinking && !cfg.ReasoningVisible() {
		return false
	}
	if b.Live {
		return true
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
	b.Live = false
}

// TruncateRunes shortens s to at most n runes with ellipsis.
func TruncateRunes(s string, n int) string {
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
