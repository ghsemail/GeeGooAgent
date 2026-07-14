package chatui

import (
	"fmt"
	"strings"
	"time"
)

// StatusBarOptions feeds the Hermes-style footer status line.
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

func renderProgressBar(pct float64, barWidth int) string {
	if barWidth < 8 {
		barWidth = 8
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	filled := int(pct*float64(barWidth) + 0.5)
	if filled > barWidth {
		filled = barWidth
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"
}

// RenderHermesStatusBar returns the fixed footer line (model · tokens · bar · timing).
func RenderHermesStatusBar(opts StatusBarOptions, width int) string {
	modelShort := strings.TrimSpace(opts.Model)
	if i := strings.LastIndex(modelShort, "/"); i >= 0 {
		modelShort = modelShort[i+1:]
	}
	if len(modelShort) > 24 {
		modelShort = modelShort[:21] + "..."
	}

	ctx := opts.ContextWindow
	if ctx <= 0 {
		ctx = 128_000
	}
	used := opts.PromptTokens
	if used < 0 {
		used = 0
	}
	pct := float64(used) / float64(ctx)
	if pct > 1 {
		pct = 1
	}

	barW := 10
	if width > 100 {
		barW = 12
	}

	elapsed := opts.Elapsed.Round(time.Second)
	if elapsed < 0 {
		elapsed = 0
	}

	busyPart := styleOK.Render(fmt.Sprintf("✓ %ds", int(elapsed.Seconds())))
	if opts.Busy {
		busyPart = styleDim.Render(fmt.Sprintf("⏲ %ds", int(elapsed.Seconds())))
	}

	parts := []string{
		styleGold.Render("⚕") + " " + styleGold.Render(modelShort),
		styleDim.Render("│"),
		styleDim.Render(FormatTokenCount(used)) + "/" + styleDim.Render(FormatTokenCount(ctx)),
		styleDim.Render(renderProgressBar(pct, barW)),
		styleDim.Render(fmt.Sprintf("%d%%", int(pct*100+0.5))),
		styleDim.Render(fmt.Sprintf("%ds", int(elapsed.Seconds()))),
		busyPart,
	}
	if opts.Steps > 0 {
		parts = append(parts, styleDim.Render(fmt.Sprintf("%d steps", opts.Steps)))
	}
	line := strings.Join(parts, " ")
	if width > 0 && len(line) > width {
		// Trim middle token segment on narrow terminals.
		parts = []string{
			styleGold.Render("⚕") + " " + styleGold.Render(modelShort),
			styleDim.Render(FormatTokenCount(used)) + "/" + styleDim.Render(FormatTokenCount(ctx)),
			styleDim.Render(fmt.Sprintf("%d%%", int(pct*100+0.5))),
			busyPart,
		}
		line = strings.Join(parts, " ")
	}
	return line
}
