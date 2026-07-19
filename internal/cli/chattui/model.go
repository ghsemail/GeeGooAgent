package chattui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// Model is the Bubble Tea root model for geegoo chat TUI.
type Model struct {
	width  int
	height int

	display config.DisplayConfig
	input   textinput.Model
	vp      viewport.Model

	slots  []*LiveSlot
	active int

	banner     string
	bannerOpts chatui.BannerOptions

	info string

	quitting bool

	configPath string
	appFactory SessionFactory

	approvalPending bool
	approvalTool    string
	approvalArgs    string

	clarifyPending      bool
	clarifyAwaitingText bool
	clarifyQuestion     string
	clarifyChoices      []string
	clarifyFocus        int

	sessionPicker bool
	pickerFocus   int

	slashPick int

	scrollFollow bool
}

// SessionFactory creates a new live Repl slot (injected by Run).
type SessionFactory func(sessionID string) (*LiveSlot, error)

// NewModel builds the initial TUI model with one slot.
func NewModel(display config.DisplayConfig, first *LiveSlot, factory SessionFactory) Model {
	display.Normalize()
	ti := textinput.New()
	ti.Placeholder = "Type your message or /help for commands."
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 60
	chatui.ConfigureTextInput(&ti)
	vp := viewport.New(80, 20)
	vp.MouseWheelEnabled = true
	m := Model{
		display:      display,
		input:        ti,
		vp:           vp,
		slots:        []*LiveSlot{first},
		active:       0,
		appFactory:   factory,
		scrollFollow: true,
	}
	if first != nil && first.Status == "" {
		first.Status = "ready"
	}
	return m
}

func (m *Model) activeSlot() *LiveSlot {
	if m.active < 0 || m.active >= len(m.slots) {
		return nil
	}
	return m.slots[m.active]
}

func (m *Model) activeHost() *ReplHost {
	s := m.activeSlot()
	if s == nil {
		return nil
	}
	return s.Host
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickApproval(), tickStatus())
}

func tickStatus() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
	})
}

type statusTickMsg time.Time

func tickApproval() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return approvalTickMsg(t)
	})
}

type approvalTickMsg time.Time

// TurnDoneMsg is sent when an agent turn finishes.
type TurnDoneMsg struct {
	Slot        int
	Reply       string
	Err         string
	PlanPending bool
	PlanTools   []string
}

// InfoMsg shows a transient status line.
type InfoMsg struct{ Text string }

// DisplayUpdatedMsg replaces display config (e.g. after /details persist).
type DisplayUpdatedMsg struct{ Display config.DisplayConfig }

// NewSessionMsg adds a live slot.
type NewSessionMsg struct{ Slot *LiveSlot }
