package chattui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/cli/progress"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// programSink forwards Agent progress for a specific live slot.
type programSink struct {
	program *tea.Program
	slot    *LiveSlot
}

func (s *programSink) EmitProgress(event string, data map[string]any) {
	if s == nil || s.program == nil || s.slot == nil {
		return
	}
	s.program.Send(ProgressMsg{Slot: s.slot.Index, Event: event, Data: data})
}

var _ progress.Sink = (*programSink)(nil)

// ShouldUseTUI reports whether interactive TUI should launch.
func ShouldUseTUI(cfg *config.AppConfig, forceTUI, forceCLI bool) bool {
	if forceCLI {
		return false
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_CHAT_PLAIN"))); v == "1" || v == "true" || v == "yes" {
		return false
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	if forceTUI {
		return true
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_CHAT_TUI"))); v == "0" || v == "false" || v == "no" {
		return false
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("GEEGOO_CHAT_TUI"))); v == "1" || v == "true" || v == "yes" {
		return true
	}
	iface := "tui"
	if cfg != nil {
		d := cfg.Display
		d.Normalize()
		iface = d.Interface
	}
	return iface != "cli"
}

// RunOpts configures the interactive TUI session.
type RunOpts struct {
	App        *app.App
	ConfigPath string
	SessionID  string
	DryRun     bool
}

// Run starts the Bubble Tea chat UI. Returns process exit code.
func Run(opts RunOpts) int {
	display := opts.App.Config.Display
	display.Normalize()

	var program *tea.Program

	makeSlot := func(sessionID string) (*LiveSlot, error) {
		repl, err := chatrepl.NewWithSession(opts.App, opts.ConfigPath, sessionID, opts.DryRun, os.Stdout)
		if err != nil {
			return nil, err
		}
		slot := &LiveSlot{
			ID:       repl.Chat.ID,
			Title:    repl.Chat.Title,
			Repl:     repl,
			SubmitCh: make(chan string, 1),
			CancelCh: make(chan struct{}, 1),
			Status:   "ready",
			Focus:    -1,
		}
		slot.Host = NewReplHost(repl, opts.ConfigPath)
		syncSlotPlanFromRepl(slot, repl)
		sink := &programSink{program: program, slot: slot}
		repl.SetProgressSink(sink)
		go runTurnHostSlot(slot, func() *tea.Program { return program })
		return slot, nil
	}

	first, err := makeSlot(opts.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat tui: %v\n", err)
		return 2
	}
	first.Index = 0

	model := NewModel(display, first, makeSlot)
	model.bannerOpts = bannerOptsFromRepl(first.Repl)
	if model.width <= 0 {
		model.width = 80
	}
	model.rebuildBanner()
	model.refreshViewport()
	model.configPath = opts.ConfigPath
	program = tea.NewProgram(model, ProgramOptions(display.MouseTracking, display.AltScreenEnabled())...)
	// Re-bind sink program pointer (created before program existed)
	if sink, ok := first.Repl.Progress.(*programSink); ok {
		sink.program = program
	} else {
		first.Repl.SetProgressSink(&programSink{program: program, slot: first})
	}

	finalModel, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat tui: %v\n", err)
		closeAllSlots([]*LiveSlot{first})
		return 1
	}
	if fm, ok := finalModel.(Model); ok {
		closeAllSlots(fm.slots)
	} else {
		closeAllSlots([]*LiveSlot{first})
	}
	return 0
}

func closeAllSlots(slots []*LiveSlot) {
	for _, s := range slots {
		if s == nil {
			continue
		}
		if s.SubmitCh != nil {
			close(s.SubmitCh)
		}
		if s.Repl != nil {
			s.Repl.CloseSession()
		}
	}
}

func runTurnHostSlot(slot *LiveSlot, programFn func() *tea.Program) {
	for text := range slot.SubmitCh {
		program := programFn()
		if program == nil {
			continue
		}
		done := make(chan TurnDoneMsg, 1)
		go func(t string) {
			result := slot.Repl.RunTurn(t)
			errText := ""
			if result.Failed {
				errText = result.Error
				if errText == "" {
					errText = "turn failed"
				}
			}
			planTools := []string{}
			if result.PlanPending && slot.Repl.Session != nil && slot.Repl.Session.PendingPlan != nil {
				planTools = toolNamesFromCalls(slot.Repl.Session.PendingPlan.ToolCalls)
			}
			done <- TurnDoneMsg{
				Slot: slot.Index, Reply: result.AssistantText, Err: errText,
				PlanPending: result.PlanPending, PlanTools: planTools,
			}
		}(text)

		select {
		case msg := <-done:
			program.Send(msg)
		case <-slot.CancelCh:
			slot.Repl.CancelTurn()
			program.Send(<-done)
		}
	}
}
