package chattui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
)

var (
	styleSessionTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorText))
	styleSessionDim   = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorDim))
	styleSessionFocus = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorGold)).Bold(true)
	styleSessionMark  = lipgloss.NewStyle().Foreground(lipgloss.Color(chatui.ColorGold))
)

// LiveSlot is one in-process chat attachment (Hermes live session).
type LiveSlot struct {
	Index    int
	ID       string
	Title    string
	Repl     *chatrepl.Repl
	Host     *ReplHost
	SubmitCh chan string
	CancelCh chan struct{}

	Blocks         []Block
	Focus          int
	Busy           bool
	Status         string
	Err            string
	Seq            int
	LiveThinkingID string
	LiveToolsID    string
	LiveReplyID    string

	TurnStartedAt time.Time
	TurnEndedAt   time.Time

	PlanPending bool
	PlanTools   []string
}

func (s *LiveSlot) shortTitle() string {
	if s.Title != "" {
		return TruncateRunes(s.Title, 24)
	}
	if s.ID == "" {
		return "(new)"
	}
	if len(s.ID) > 10 {
		return s.ID[:10]
	}
	return s.ID
}

func (s *LiveSlot) statusGlyph() string {
	if s.Busy {
		return "●"
	}
	return "○"
}

func formatSessionList(slots []*LiveSlot, active, pickerFocus int) string {
	var b strings.Builder
	b.WriteString(styleSessionDim.Render("Live sessions (Enter 切换 · Ctrl+N 新建 · Ctrl+D 关闭 · Esc 取消)"))
	b.WriteByte('\n')
	for i, s := range slots {
		mark := "  "
		if i == pickerFocus {
			mark = styleSessionMark.Render("› ")
		}
		cur := ""
		if i == active {
			cur = styleSessionFocus.Render(" [active]")
		}
		busy := ""
		if s.Busy {
			busy = styleSessionDim.Render(" running")
		}
		glyph := styleSessionDim.Render(s.statusGlyph())
		title := styleSessionTitle.Render(s.shortTitle())
		if i == pickerFocus {
			title = styleSessionFocus.Render(s.shortTitle())
			glyph = styleSessionFocus.Render(s.statusGlyph())
		}
		b.WriteString(fmt.Sprintf("%s%s %s%s%s\n", mark, glyph, title, cur, busy))
	}
	b.WriteString(styleSessionDim.Render("  + new session"))
	b.WriteByte('\n')
	return b.String()
}
