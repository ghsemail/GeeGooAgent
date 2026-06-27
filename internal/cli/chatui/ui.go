package chatui

import (
	"fmt"
	"io"
	"os"
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
	u.println(styleDim.Render(strings.Repeat("─", u.ruleWidth())))
}

// PrintBanner shows Hermes-style two-column welcome panel.
func (u *ChatUI) PrintBanner(opts BannerOptions) {
	if u.plain {
		u.println("")
		u.write(BuildPlainBanner(opts))
		return
	}
	rev := opts.Revision
	if rev == "" {
		rev = ResolveRevision(opts.InstallDir)
	}
	u.println("")
	if u.width >= 95 {
		u.println(renderWideLogo())
		u.println("")
	}
	left := buildBannerLeft(opts)
	right := buildBannerRight(opts)
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Padding(0, 2).Align(lipgloss.Center).Render(left),
		lipgloss.NewStyle().Padding(0, 1).Render(right),
	)
	u.println(styleGold.Render(formatVersionLabel(rev)))
	u.println(stylePanel.Render(cols))
	u.println("")
	u.println(styleText.Render("Welcome to GeeGoo Agent! ") + styleDim.Render("Type your message or /help for commands."))
	u.println(
		styleDim.Render("✦ Tip: ") +
			styleDim.Render("/think on") + styleDim.Render(" shows DeepSeek reasoning; ") +
			styleDim.Render("/verbose off") + styleDim.Render(" hides live steps."),
	)
	u.println("")
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
	u.println(styleGold.Render("● ") + styleText.Render(text))
}

func (u *ChatUI) PrintAssistant(text string) {
	if u.plain {
		u.println("")
		u.println(text)
		u.println("")
		return
	}
	body := text
	if r, err := newMarkdownRenderer(u.contentWidth()); err == nil {
		if rendered, err := r.Render(text); err == nil {
			body = strings.TrimRight(rendered, "\n")
		}
	}
	title := styleGold.Render("⚕ GeeGoo")
	inner := title + "\n" + body
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 1).
		Render(inner)
	u.println(box)
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
		if u.plain {
			u.println("────────────────")
			u.println("Initializing agent...")
			return
		}
		u.printRule()
		u.println(styleDim.Render("Initializing agent..."))
	case "round_start":
		if u.plain {
			u.println(fmt.Sprintf("⋯ step %v", data["step"]))
		}
	case "llm_plan":
		reasoning, _ := data["reasoning"].(string)
		content, _ := data["content"].(string)
		toolNames, _ := data["tool_names"].([]string)
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
					u.println("    " + styleDim.Render(line))
				}
			}
		}
		if strings.TrimSpace(content) != "" {
			u.println("  " + styleAmber.Render("📋 计划") + " " + styleText.Render(truncate(content, 280)))
		}
		if len(toolNames) > 0 {
			u.println("  " + styleDim.Render("→ 调用: "+strings.Join(toolNames, ", ")))
		}
	case "llm_tools":
		if u.plain {
			if names, ok := data["tool_names"].([]string); ok {
				u.println(fmt.Sprintf("  ⋯ 计划调用: %s", strings.Join(names, ", ")))
			}
			return
		}
		if names, ok := data["tool_names"].([]string); ok && len(names) > 0 {
			u.println("  " + styleDim.Render("⋯ 执行: "+strings.Join(names, ", ")))
		}
	case "tool_start":
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
			styleDim.Render(argsPart),
			styleDim.Render(fmt.Sprintf("%.1fs", duration)),
		))
	case "error":
		msg, _ := data["message"].(string)
		u.PrintError(msg)
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
