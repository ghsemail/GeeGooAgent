package chattui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
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
	b.WriteString("Live sessions (Enter 切换 · Ctrl+N 新建 · Ctrl+D 关闭 · Esc 取消)\n")
	for i, s := range slots {
		mark := "  "
		if i == pickerFocus {
			mark = "› "
		}
		cur := ""
		if i == active {
			cur = " [active]"
		}
		busy := ""
		if s.Busy {
			busy = " running"
		}
		b.WriteString(fmt.Sprintf("%s%s %s%s%s\n", mark, s.statusGlyph(), s.shortTitle(), cur, busy))
	}
	b.WriteString("  + new session\n")
	return b.String()
}
