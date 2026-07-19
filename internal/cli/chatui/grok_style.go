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
			lines = append(lines, styleBrand.Render("> ")+styleUser.Render(strings.TrimPrefix(line, "> ")))
			continue
		}
		lines = append(lines, styleUser.Render(line))
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
	return styleMeta.Render("Working…")
}

// RenderTurnFooter renders turn timing after a completed reply.
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
	return styleWhisper.Render(label)
}

// RenderInputChrome wraps the live input line in a bordered box.
func RenderInputChrome(inputLine string, model string, width int) string {
	box := styleInputBox
	if width > 0 {
		box = box.Width(width)
	}
	prompt := styleBrand.Render("> ") + inputLine
	inner := prompt
	model = strings.TrimSpace(model)
	if model != "" {
		if i := strings.LastIndex(model, "/"); i >= 0 {
			model = model[i+1:]
		}
		if len(model) > 28 {
			model = model[:25] + "..."
		}
		modelLine := styleMeta.Render(model)
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
	body.WriteString(styleTitle.Render("GeeGoo Agent"))
	body.WriteByte('\n')
	body.WriteString(styleMeta.Render(fmt.Sprintf("%s · %s / %s", formatVersionLabel(rev), opts.Provider, model)))
	body.WriteByte('\n')
	body.WriteString(styleWhisper.Render("Type a message or /help for commands."))
	box := styleInputBox.Width(width)
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

// AnchorContentBottomKeepingPrefix keeps a leading section (e.g. welcome banner) at the
// top of the viewport and only pads the remainder so short conversations sit above the input.
func AnchorContentBottomKeepingPrefix(prefix, content string, viewportHeight int) string {
	if viewportHeight <= 0 || strings.TrimSpace(content) == "" {
		return content
	}
	head := ""
	tail := content
	if prefix != "" && strings.HasPrefix(content, prefix) {
		head = prefix
		tail = content[len(prefix):]
	}
	if strings.TrimSpace(tail) == "" {
		return head
	}
	headLines := 0
	if head != "" {
		headLines = strings.Count(head, "\n") + 1
	}
	avail := viewportHeight - headLines
	if avail < 1 {
		avail = 1
	}
	return head + AnchorContentBottom(tail, avail)
}

// RenderGrokProcessHeader renders a section header (thinking / tools).
func RenderGrokProcessHeader(expanded bool, title string, lineCount int, durationSec float64) string {
	chevron := "▸"
	if expanded {
		chevron = "▾"
	}
	meta := ""
	if lineCount > 0 {
		meta = fmt.Sprintf(" · %d 行", lineCount)
	}
	if durationSec > 0 {
		meta += fmt.Sprintf(" · %.1fs", durationSec)
	}
	return styleMeta.Render(chevron+" ") + styleSection.Render(title) + styleWhisper.Render(meta)
}

// RenderGrokThinkingLine renders a thinking detail line.
func RenderGrokThinkingLine(line string, width int) string {
	innerW := ContentWrapWidth(width) - 2
	if innerW < 24 {
		innerW = 24
	}
	var b strings.Builder
	for i, wl := range strings.Split(WrapPlain(line, innerW), "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("  ")
		b.WriteString(styleThinking.Render(wl))
	}
	return b.String()
}

// RenderGrokToolLine renders a tool/activity detail line (sidebar + plain text).
func RenderGrokToolLine(line string, width int) string {
	innerW := ContentWrapWidth(width) - 4
	if innerW < 24 {
		innerW = 24
	}
	var b strings.Builder
	trim := strings.TrimSpace(line)
	if trim == "" {
		return ""
	}
	prefix := "  "
	sidebar := styleWhisper.Render("│ ")
	for i, wl := range strings.Split(WrapPlain(trim, innerW), "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == 0 {
			b.WriteString(prefix + sidebar + styleTool.Render(wl))
			continue
		}
		b.WriteString(prefix + strings.Repeat(" ", lipgloss.Width(sidebar)) + styleTool.Render(wl))
	}
	return b.String()
}

// RenderGrokReplyBlock renders assistant reply body with clean paragraph spacing.
func RenderGrokReplyBlock(text string, width int) string {
	body := RenderPlainAssistantBody(text, assistantWrapWidth(width))
	if body == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	var out []string
	for i, line := range lines {
		out = append(out, line)
		if line == "" || i == len(lines)-1 {
			continue
		}
		next := strings.TrimSpace(lines[i+1])
		cur := strings.TrimSpace(stripANSI(line))
		if cur != "" && next != "" && !strings.HasPrefix(next, "  ") {
			out = append(out, "")
		}
	}
	return strings.Join(out, "\n")
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
