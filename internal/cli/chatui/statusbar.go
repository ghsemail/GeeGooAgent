package chatui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StatusBarOptions feeds the Grok-style footer status line.
type StatusBarOptions struct {
	Model         string
	PromptTokens  int
	ContextWindow int
	Elapsed       time.Duration
	Busy          bool
	Steps         int
}

// FormatTokenCount abbreviates token counts (e.g. 19600 → 19.6K).
func FormatTokenCount(n int) string {
	if n <= 0 {
		return "0"
	}
	switch {
	case n >= 1_000_000:
		v := float64(n) / 1_000_000
		if v >= 10 {
			return fmt.Sprintf("%.0fM", v)
		}
		return fmt.Sprintf("%.1fM", v)
	case n >= 1000:
		v := float64(n) / 1000
		if v >= 100 || v == float64(int(v)) {
			return fmt.Sprintf("%.0fK", v)
		}
		return fmt.Sprintf("%.1fK", v)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// RenderGrokFooterBar returns hints on the left and token usage on the right.
func RenderGrokFooterBar(opts StatusBarOptions, width int) string {
	left := styleDim.Render("Tab: sessions · /help")
	ctx := opts.ContextWindow
	if ctx <= 0 {
		ctx = 128_000
	}
	used := opts.PromptTokens
	if used < 0 {
		used = 0
	}
	right := styleDim.Render(FormatTokenCount(used) + " / " + FormatTokenCount(ctx))
	if width <= 0 {
		return left + "  " + right
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// RenderHermesStatusBar is kept for compatibility; delegates to Grok footer bar.
func RenderHermesStatusBar(opts StatusBarOptions, width int) string {
	return RenderGrokFooterBar(opts, width)
}
