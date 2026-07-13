package chatui

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ChatUI renders Hermes / Claude Code inspired terminal output.
type ChatUI struct {
	out   io.Writer
	plain bool
	width int

	toolStarts map[string]time.Time
	toolArgs   map[string]string

	streamActive   bool
	streamBuf      strings.Builder
	replyStreamed  bool
	streamRoundHad bool // content streamed in current LLM round (before tools)
}

// New creates a ChatUI; plain when not a TTY or GEEGOO_CHAT_PLAIN=1.
func New(out io.Writer) *ChatUI {
	if out == nil {
		out = os.Stdout
	}
	plain := true
	width := 80
	if f, ok := out.(*os.File); ok {
		if fi, err := f.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			plain = false
			if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
				width = w
			}
		}
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_CHAT_PLAIN"))); v == "1" || v == "true" || v == "yes" {
		plain = true
	}
	return &ChatUI{
		out: out, plain: plain, width: width,
		toolStarts: make(map[string]time.Time),
		toolArgs:   make(map[string]string),
	}
}

func (u *ChatUI) IsPlain() bool { return u.plain }

func (u *ChatUI) write(text string) {
	_, _ = io.WriteString(u.out, text)
}

func (u *ChatUI) println(text string) {
	u.write(text)
	if !strings.HasSuffix(text, "\n") {
		u.write("\n")
	}
}

func (u *ChatUI) contentWidth() int {
	w := u.width - 4
	if w < 40 {
		return 40
	}
	if w > 100 {
		return 100
	}
	return w
}

func (u *ChatUI) ruleWidth() int {
	w := u.width
	if w < 40 {
		return 40
	}
	if w > 80 {
		return 80
	}
	return w
}

func (u *ChatUI) printRule() {
	u.println(RenderRule(u.width))
}

// RenderRule returns a Hermes-style horizontal rule.
func RenderRule(width int) string {
	w := width
	if w < 40 {
		w = 40
	}
	if w > 80 {
		w = 80
	}
	return styleDim.Render(strings.Repeat("─", w))
}

// RenderInitializing returns the turn-start status line.
func RenderInitializing() string {
	return styleDim.Render("Initializing agent...")
}

// RenderThinkingLine returns body text for expanded reasoning sections.
func RenderThinkingLine(line string) string {
	return styleText.Render("    " + line)
}

// RenderDetailLine returns body text for tool/activity detail sections.
func RenderDetailLine(line string) string {
	return styleText.Render("    " + line)
}

// PrintBanner shows Hermes-style two-column welcome panel.
func (u *ChatUI) PrintBanner(opts BannerOptions) {
	u.write(RenderBanner(opts, u.width, u.plain))
}

// RenderBanner returns the Hermes-style welcome panel as a string (CLI and TUI).
func RenderBanner(opts BannerOptions, width int, plain bool) string {
	if width <= 0 {
		width = 80
	}
	if plain {
		return "\n" + BuildPlainBanner(opts)
	}
	rev := opts.Revision
	if rev == "" {
		rev = ResolveRevision(opts.InstallDir)
	}
	var b strings.Builder
	b.WriteByte('\n')
	if width >= 95 {
		b.WriteString(renderWideLogo())
		b.WriteByte('\n')
		b.WriteByte('\n')
	}
	left := buildBannerLeft(opts)
	right := buildBannerRight(opts)
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Padding(0, 2).Align(lipgloss.Center).Render(left),
		lipgloss.NewStyle().Padding(0, 1).Render(right),
	)
	b.WriteString(styleGold.Render(formatVersionLabel(rev)))
	b.WriteByte('\n')
	b.WriteString(stylePanel.Render(cols))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(styleText.Render("Welcome to GeeGoo Agent! ") + styleDim.Render("Type your message or /help for commands."))
	b.WriteByte('\n')
	b.WriteString(RenderWelcomeTips())
	b.WriteByte('\n')
	b.WriteByte('\n')
	return b.String()
}

// RenderAssistantBox returns a Hermes-style rounded reply panel with glamour markdown.
func RenderAssistantBox(text string, width int) string {
	return renderAssistantPanel(text, width, false)
}

// RenderAssistantBoxLive shows a streaming preview without glamour (avoids broken partial tables).
func RenderAssistantBoxLive(text string, width int) string {
	return renderAssistantPanel(text, width, true)
}

func renderAssistantPanel(text string, width int, live bool) string {
	if width <= 0 {
		width = 80
	}
	contentW := assistantContentWidth(width)
	body := strings.TrimRight(text, "\n")
	if live {
		body = RenderPlainAssistantBody(body)
	} else {
		body = renderAssistantMarkdown(body, contentW)
	}
	title := styleGold.Render("⚕ GeeGoo")
	inner := title + "\n" + body
	// Do not set Width() here: lipgloss reflow would collapse glamour/plain newlines into one paragraph.
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 1).
		MaxWidth(width - 2).
		Render(inner)
}

func assistantContentWidth(width int) int {
	contentW := width - 6
	if contentW < 40 {
		return 40
	}
	if contentW > 120 {
		return 120
	}
	return contentW
}

func renderAssistantMarkdown(text string, contentW int) string {
	_ = contentW
	return RenderPlainAssistantBody(text)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RenderUserLine returns Hermes-style user message bullet.
func RenderUserLine(text string) string {
	return styleGold.Render("● ") + styleText.Render(text)
}

func (u *ChatUI) PrintStatusBar(model string, thinking, dryRun bool, steps int) {
	if u.plain {
		return
	}
	modelShort := model
	if i := strings.LastIndex(model, "/"); i >= 0 {
		modelShort = model[i+1:]
	}
	if len(modelShort) > 20 {
		modelShort = modelShort[:17] + "..."
	}
	think := "off"
	if thinking {
		think = "on"
	}
	dry := "off"
	if dryRun {
		dry = "on"
	}
	u.println(fmt.Sprintf(" %s %s %s think %s │ dry-run %s │ %d steps",
		styleGold.Render("⚕"),
		styleGold.Render(modelShort),
		styleDim.Render("│"),
		think, dry, steps,
	))
}

func (u *ChatUI) PrintTurnFooter(model string, thinking, dryRun bool, steps int) {
	if u.plain {
		u.println("")
		return
	}
	u.PrintStatusBar(model, thinking, dryRun, steps)
	u.printRule()
}

func (u *ChatUI) PrintPrompt() {
	if u.plain {
		u.write("> ")
		return
	}
	u.write(styleGold.Render("❯ "))
}

func (u *ChatUI) PrintUser(text string) {
	if u.plain {
		u.println(fmt.Sprintf("\n> %s\n", text))
		return
	}
	u.println("")
	u.println(RenderUserLine(text))
}

func (u *ChatUI) PrintAssistant(text string) {
	if u.plain {
		u.println("")
		u.println(text)
		u.println("")
		return
	}
	u.println(RenderAssistantBox(text, u.width))
}

// ResetStream clears typewriter state at the start of a user turn.
func (u *ChatUI) ResetStream() {
	u.streamActive = false
	u.streamBuf.Reset()
	u.replyStreamed = false
	u.streamRoundHad = false
}

// WriteStreamDelta appends live assistant text (typewriter).
func (u *ChatUI) WriteStreamDelta(text string) {
	text = stripStreamNoise(text)
	if strings.TrimSpace(text) == "" {
		return
	}
	if !u.streamActive {
		u.streamActive = true
		u.streamBuf.Reset()
		u.println("")
		if u.plain {
			u.write("GeeGoo> ")
		}
	}
	u.streamBuf.WriteString(text)
	u.streamRoundHad = true
	if u.plain {
		u.write(text)
	}
}

// AbortStreamReply discards in-progress streamed text when tool calls follow
// (that content was plan/thinking, not the final user-facing reply).
func (u *ChatUI) AbortStreamReply() {
	if !u.streamActive {
		u.streamRoundHad = false
		return
	}
	hadVisible := strings.TrimSpace(u.streamBuf.String()) != ""
	u.write("\n")
	if hadVisible && !u.plain {
		u.println(styleText.Render("  ↳ （计划文本，继续调用工具…）"))
	}
	u.streamActive = false
	u.streamBuf.Reset()
	u.streamRoundHad = false
}

var streamSIDRE = regexp.MustCompile(`(?i)\[SID=[^\]]+\]`)

func stripStreamNoise(s string) string {
	return streamSIDRE.ReplaceAllString(s, "")
}

// FinishAssistantStream closes a live reply stream. Returns true when the
// final answer was already printed (caller should skip PrintAssistant).
func (u *ChatUI) FinishAssistantStream() bool {
	if u.streamActive {
		u.streamActive = false
		if u.streamBuf.Len() > 0 {
			final := strings.TrimRight(u.streamBuf.String(), "\n")
			if u.plain {
				u.write("\n")
				u.println("")
			} else {
				u.println(RenderAssistantBox(final, u.width))
			}
			u.replyStreamed = true
		} else if !u.plain {
			u.println("")
		}
	}
	return u.replyStreamed
}

// StreamRoundHadContent reports whether the current LLM round already streamed
// visible content (used to avoid duplicating llm_plan text).
func (u *ChatUI) StreamRoundHadContent() bool {
	return u.streamRoundHad
}

func (u *ChatUI) swapWriter(w io.Writer) func() {
	old := u.out
	u.out = w
	return func() { u.out = old }
}

func (u *ChatUI) setPlain(v bool) bool {
	old := u.plain
	u.plain = v
	return old
}

// WithPlainWriter temporarily sends Print* output to w as plain text.
func (u *ChatUI) WithPlainWriter(w io.Writer, fn func()) {
	restore := u.swapWriter(w)
	wasPlain := u.setPlain(true)
	fn()
	u.setPlain(wasPlain)
	restore()
}

// RunPlainCapture redirects UI output to a plain-text buffer while fn runs.
func (u *ChatUI) RunPlainCapture(fn func()) string {
	var buf strings.Builder
	u.WithPlainWriter(&buf, fn)
	return strings.TrimSpace(buf.String())
}

func (u *ChatUI) PrintHelp(text string) {
	if u.plain {
		u.write(text)
		return
	}
	u.println(styleDim.Render(strings.TrimSpace(text)))
}

func (u *ChatUI) PrintInfo(text string) {
	if u.plain {
		u.println(text)
		return
	}
	u.println(styleDim.Render(text))
}

func (u *ChatUI) PrintError(text string) {
	if u.plain {
		u.println("✗ " + text)
		return
	}
	u.println(styleErr.Render("✗ " + text))
}

func (u *ChatUI) toolKey(data map[string]any) string {
	return fmt.Sprintf("%v:%v", data["step"], data["name"])
}

// EmitProgress handles ReAct live events (verbose mode).
func (u *ChatUI) EmitProgress(event string, data map[string]any) {
	switch event {
	case "turn_start":
		u.ResetStream()
		if u.plain {
			u.println("────────────────")
			u.println("Initializing agent...")
			return
		}
		u.printRule()
		u.println(styleDim.Render("Initializing agent..."))
	case "round_start":
		u.streamRoundHad = false
		if u.plain {
			u.println(fmt.Sprintf("⋯ step %v", data["step"]))
		}
	case "stream_delta":
		if content, _ := data["content"].(string); content != "" {
			u.WriteStreamDelta(content)
		}
	case "llm_plan":
		reasoning, _ := data["reasoning"].(string)
		content, _ := data["content"].(string)
		toolNames, _ := data["tool_names"].([]string)
		if u.StreamRoundHadContent() {
			content = "" // already shown via typewriter
		}
		if u.plain {
			if strings.TrimSpace(reasoning) != "" {
				u.println(fmt.Sprintf("  [思考] %s", truncate(reasoning, 500)))
			}
			if strings.TrimSpace(content) != "" {
				u.println(fmt.Sprintf("  [计划] %s", truncate(content, 300)))
			}
			if len(toolNames) > 0 {
				u.println(fmt.Sprintf("  [决策] 调用: %s", strings.Join(toolNames, ", ")))
			}
			return
		}
		if strings.TrimSpace(reasoning) != "" {
			u.println("  " + styleAmber.Render("💭 思考"))
			for _, line := range strings.Split(truncate(reasoning, 600), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					u.println("    " + styleText.Render(line))
				}
			}
		}
		if strings.TrimSpace(content) != "" {
			u.println("  " + styleAmber.Render("📋 计划") + " " + styleText.Render(truncate(content, 280)))
		}
		if len(toolNames) > 0 {
			u.println("  " + styleText.Render("→ 调用: "+strings.Join(toolNames, ", ")))
		}
	case "llm_tools":
		u.AbortStreamReply()
		if u.plain {
			if names, ok := data["tool_names"].([]string); ok {
				u.println(fmt.Sprintf("  ⋯ 计划调用: %s", strings.Join(names, ", ")))
			}
			return
		}
		if names, ok := data["tool_names"].([]string); ok && len(names) > 0 {
			u.println("  " + styleText.Render("⋯ 执行: "+strings.Join(names, ", ")))
		}
	case "tool_start":
		u.AbortStreamReply()
		name, _ := data["name"].(string)
		key := u.toolKey(data)
		u.toolStarts[key] = time.Now()
		args, _ := data["arguments"].(map[string]any)
		u.toolArgs[key] = fmtArgsCompact(args)
		if u.plain {
			u.println(fmt.Sprintf("  → %s(%s)", name, fmtArgs(args)))
			return
		}
		label := displayToolName(name)
		u.println(fmt.Sprintf("  ┊ %s %s",
			toolEmoji(name), styleToolRun.Render("preparing "+label+"…")))
	case "tool_done":
		name, _ := data["name"].(string)
		status, _ := data["status"].(string)
		key := u.toolKey(data)
		started := u.toolStarts[key]
		if started.IsZero() {
			started = time.Now()
		}
		delete(u.toolStarts, key)
		args := u.toolArgs[key]
		delete(u.toolArgs, key)
		duration := time.Since(started).Seconds()
		if u.plain {
			mark := "✓"
			if status != string(tools.StatusOK) {
				mark = "✗"
			}
			summary, _ := data["summary"].(string)
			u.println(fmt.Sprintf("  %s %s [%s] %s", mark, name, status, truncate(summary, 160)))
			return
		}
		ok := status == string(tools.StatusOK)
		color := styleGold
		if !ok {
			color = styleErr
		}
		argsPart := ""
		if args != "" {
			argsPart = "  " + args
		}
		label := displayToolName(name)
		u.println(fmt.Sprintf("  ┊ %s %s%s %s",
			toolEmoji(name),
			color.Render(fmt.Sprintf("%-22s", label)),
			styleText.Render(argsPart),
			styleThinking.Render(fmt.Sprintf("%.1fs", duration)),
		))
	case "error":
		u.AbortStreamReply()
		msg, _ := data["message"].(string)
		u.PrintError(msg)
	case "reply_start":
		// Final reply may already be streaming; nothing else to do.
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func fmtArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, v := range args {
		parts = append(parts, fmt.Sprint(v))
	}
	return truncate(strings.Join(parts, "  "), 72)
}

func fmtArgsCompact(args map[string]any) string {
	return fmtArgs(args)
}
