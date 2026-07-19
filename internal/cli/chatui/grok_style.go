package chatui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RenderUserPromptBox renders a Grok-style committed user prompt (gray band + "> ").
func RenderUserPromptBox(text string, width int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	innerW := ContentWrapWidth(width)
	if innerW < 24 {
		innerW = 24
	}
	wrapped := WrapWithPrefix(text, "> ", "  ", innerW)
	var lines []string
	for _, line := range strings.Split(wrapped, "\n") {
		if strings.HasPrefix(line, "> ") {
			lines = append(lines, styleText.Render("> ")+styleText.Render(strings.TrimPrefix(line, "> ")))
			continue
		}
		lines = append(lines, styleText.Render(line))
	}
	box := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorBgPrompt)).
		Padding(0, 1)
	if width > 0 {
		box = box.Width(width)
	}
	return box.Render(strings.Join(lines, "\n"))
}

// RenderWorkingLine shows a compact in-turn status while the agent is busy.
func RenderWorkingLine() string {
	return styleDim.Render("Working…")
}

// RenderTurnFooter renders Grok-style turn timing after a completed reply.
func RenderTurnFooter(elapsed time.Duration) string {
	if elapsed < 0 {
		elapsed = 0
	}
	sec := elapsed.Seconds()
	var label string
	switch {
	case sec < 10:
		label = fmt.Sprintf("Worked for %.1fs.", sec)
	case sec < 60:
		label = fmt.Sprintf("Worked for %.0fs.", sec)
	default:
		m := int(sec) / 60
		s := int(sec) % 60
		if s == 0 {
			label = fmt.Sprintf("Worked for %dm.", m)
		} else {
			label = fmt.Sprintf("Worked for %dm%ds.", m, s)
		}
	}
	return styleDim.Render(label)
}

// RenderInputChrome wraps the live input line in a Grok-style bordered box.
// When model is non-empty it is shown right-aligned inside the box (Grok input chrome).
func RenderInputChrome(inputLine string, model string, width int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorDim)).
		Padding(0, 1)
	if width > 0 {
		box = box.Width(width)
	}
	prompt := styleText.Render("> ") + inputLine
	inner := prompt
	model = strings.TrimSpace(model)
	if model != "" {
		if i := strings.LastIndex(model, "/"); i >= 0 {
			model = model[i+1:]
		}
		if len(model) > 28 {
			model = model[:25] + "..."
		}
		modelLine := styleDim.Render(model)
		innerW := width - 4
		if innerW < 20 {
			innerW = 20
		}
		promptW := lipgloss.Width(prompt)
		modelW := lipgloss.Width(modelLine)
		gap := innerW - promptW - modelW
		if gap < 1 {
			gap = 1
		}
		inner = prompt + strings.Repeat(" ", gap) + modelLine
	}
	return box.Render(inner)
}

// RenderGrokWelcomeCard returns a compact bordered welcome panel (flush-left).
func RenderGrokWelcomeCard(opts BannerOptions, width int) string {
	if width <= 0 {
		width = 80
	}
	rev := opts.Revision
	if rev == "" {
		rev = ResolveRevision(opts.InstallDir)
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = "default"
	}
	var body strings.Builder
	body.WriteString(styleText.Render("GeeGoo Agent"))
	body.WriteByte('\n')
	body.WriteString(styleDim.Render(fmt.Sprintf("%s · %s / %s", formatVersionLabel(rev), opts.Provider, model)))
	body.WriteByte('\n')
	body.WriteString(styleDim.Render("Type a message or /help for commands."))
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorDim)).
		Padding(0, 1).
		Width(width)
	return "\n" + box.Render(body.String()) + "\n"
}

// AnchorContentBottom pads content so the last lines sit above the input when short.
func AnchorContentBottom(content string, viewportHeight int) string {
	if viewportHeight <= 0 || strings.TrimSpace(content) == "" {
		return content
	}
	lines := strings.Count(content, "\n") + 1
	if lines >= viewportHeight {
		return content
	}
	return strings.Repeat("\n", viewportHeight-lines) + content
}
