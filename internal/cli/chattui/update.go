package chattui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatcmd"
)

var (
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	styleUser   = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width > 4 {
			m.input.Width = msg.Width - 4
		}
		return m, nil

	case ProgressMsg:
		m.ApplyProgress(msg.Event, msg.Data)
		return m, nil

	case TurnDoneMsg:
		reply := strings.TrimSpace(msg.Reply)
		if msg.Err != "" {
			m.err = msg.Err
		}
		if reply != "" {
			if idx := findLastKind(m.blocks, KindReply); idx >= 0 {
				if strings.TrimSpace(m.blocks[idx].Body) == "" {
					m.blocks[idx].Body = reply
				}
			} else {
				m.blocks = append(m.blocks, Block{
					ID: fmt.Sprintf("reply-%d", m.seq), Kind: KindReply, Title: "助手", Body: reply,
				})
				m.seq++
			}
		}
		m.finalizeLiveSections()
		m.input.Focus()
		return m, nil

	case InfoMsg:
		m.info = msg.Text
		return m, nil

	case DisplayUpdatedMsg:
		m.display = msg.Display
		m.display.Normalize()
		return m, nil

	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.busy && m.cancelCh != nil {
				select {
				case m.cancelCh <- struct{}{}:
				default:
				}
				m.info = "正在中断…"
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.busy && m.cancelCh != nil {
				select {
				case m.cancelCh <- struct{}{}:
				default:
				}
				return m, nil
			}
			return m, nil
		case tea.KeyUp:
			m.moveFocus(-1)
			return m, nil
		case tea.KeyDown:
			m.moveFocus(1)
			return m, nil
		case tea.KeyEnter:
			if msg.Alt {
				// Alt+Enter: insert newline later; Phase A ignores
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.input.SetValue("")
			return m.handleSubmit(text)
		case tea.KeySpace:
			// Toggle focused block when input empty
			if strings.TrimSpace(m.input.Value()) == "" && m.focus >= 0 && m.focus < len(m.blocks) {
				m.blocks[m.focus].ToggleExpand(m.display)
				return m, nil
			}
		}
	}

	if !m.busy {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) moveFocus(delta int) {
	if len(m.blocks) == 0 {
		m.focus = -1
		return
	}
	if m.focus < 0 {
		if delta > 0 {
			m.focus = 0
		} else {
			m.focus = len(m.blocks) - 1
		}
		return
	}
	m.focus += delta
	if m.focus < 0 {
		m.focus = 0
	}
	if m.focus >= len(m.blocks) {
		m.focus = len(m.blocks) - 1
	}
}

func (m Model) handleSubmit(text string) (tea.Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		return m.handleSlash(text)
	}
	if m.busy {
		m.info = "请等待当前回合结束，或 Esc 中断"
		return m, nil
	}
	m.err = ""
	m.info = ""
	m.blocks = append(m.blocks, Block{
		ID: fmt.Sprintf("user-%d", m.seq), Kind: KindUser, Title: "你", Body: text,
	})
	m.seq++
	m.busy = true
	m.status = "thinking…"
	if m.submitCh != nil {
		go func() { m.submitCh <- text }()
	}
	return m, nil
}

func (m Model) handleSlash(text string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(text)
	cmd := strings.ToLower(fields[0])
	args := fields[1:]
	switch cmd {
	case "/exit", "/quit":
		m.quitting = true
		return m, tea.Quit
	case "/details":
		res := chatcmd.ApplyDetails(&m.display, args)
		if !res.OK {
			m.info = res.Message
			return m, nil
		}
		if res.Display != nil {
			m.display = *res.Display
		}
		if res.ShowLast {
			m.expandLastDetails()
		}
		m.info = res.Message
		if res.Persist {
			cp := m.configPath
			disp := m.display
			return m, func() tea.Msg {
				if cp != "" {
					_ = PersistDisplay(cp, disp)
				}
				return DisplayUpdatedMsg{Display: disp}
			}
		}
		return m, nil
	case "/help":
		m.info = "Space 折叠焦点块 · ↑/↓ 选块 · /details · /exit · Esc 中断"
		return m, nil
	default:
		m.info = "TUI Phase A 支持: /help /details /exit（其余命令请用 geegoo chat --cli）"
		return m, nil
	}
}

func (m Model) View() string {
	var b strings.Builder
	title := styleAccent.Render("⚕ GeeGoo Chat TUI")
	b.WriteString(title)
	b.WriteString(styleDim.Render("  "+m.status))
	if m.info != "" {
		b.WriteString("\n")
		b.WriteString(styleDim.Render(m.info))
	}
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(styleErr.Render(m.err))
	}
	b.WriteString("\n")
	b.WriteString(styleDim.Render(strings.Repeat("─", max(20, m.width-1))))
	b.WriteString("\n")

	for i, block := range m.blocks {
		if !block.IsVisible(m.display) {
			continue
		}
		prefix := "  "
		if i == m.focus {
			prefix = styleOK.Render("› ")
		}
		switch block.Kind {
		case KindUser:
			b.WriteString(prefix + styleUser.Render("你") + " " + block.Body + "\n")
		case KindReply:
			b.WriteString(prefix + styleAccent.Render("助手") + "\n")
			for _, line := range strings.Split(block.Body, "\n") {
				b.WriteString("    " + line + "\n")
			}
		default:
			b.WriteString(prefix + headerLabel(block, m.display) + "\n")
			if block.IsExpanded(m.display) {
				body := strings.TrimRight(block.Body, "\n")
				for _, line := range strings.Split(body, "\n") {
					b.WriteString(styleDim.Render("    "+line) + "\n")
				}
			}
		}
	}

	b.WriteString(styleDim.Render(strings.Repeat("─", max(20, m.width-1))))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	return b.String()
}

func findLastKind(blocks []Block, kind SectionKind) int {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Kind == kind {
			return i
		}
	}
	return -1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure textinput import used when Blink etc.
var _ = textinput.New
