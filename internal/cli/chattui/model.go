package chattui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// Model is the Bubble Tea root model for geegoo chat TUI.
type Model struct {
	width  int
	height int

	display config.DisplayConfig
	blocks  []Block
	focus   int
	input   textinput.Model

	busy   bool
	status string
	info   string
	err    string

	seq            int
	liveThinkingID string
	liveToolsID    string
	liveReplyID    string
	turnEnded      time.Time

	quitting bool
	ready    bool

	// submitCh receives user prompts; host goroutine runs agent turns.
	submitCh chan string
	cancelCh chan struct{}

	configPath string
	host       *ReplHost

	approvalPending bool
	approvalTool    string
	approvalArgs    string
}

// NewModel builds the initial TUI model.
func NewModel(display config.DisplayConfig, submitCh chan string, cancelCh chan struct{}) Model {
	display.Normalize()
	ti := textinput.New()
	ti.Placeholder = "输入问题，或 /details /exit …"
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 60
	return Model{
		display:  display,
		input:    ti,
		status:   "ready",
		submitCh: submitCh,
		cancelCh: cancelCh,
		focus:    -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickApproval())
}

func tickApproval() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return approvalTickMsg(t)
	})
}

type approvalTickMsg time.Time

// TurnDoneMsg is sent when an agent turn finishes.
type TurnDoneMsg struct {
	Reply string
	Err   string
}

// InfoMsg shows a transient status line.
type InfoMsg struct{ Text string }

// DisplayUpdatedMsg replaces display config (e.g. after /details persist).
type DisplayUpdatedMsg struct{ Display config.DisplayConfig }
