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

// programSink forwards Agent progress into a running tea.Program.
type programSink struct {
	program *tea.Program
}

func (s *programSink) EmitProgress(event string, data map[string]any) {
	if s == nil || s.program == nil {
		return
	}
	s.program.Send(ProgressMsg{Event: event, Data: data})
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
	repl, err := chatrepl.NewWithSession(opts.App, opts.ConfigPath, opts.SessionID, opts.DryRun, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chat tui: %v\n", err)
		return 2
	}

	submitCh := make(chan string, 1)
	cancelCh := make(chan struct{}, 1)
	display := opts.App.Config.Display
	display.Normalize()

	model := NewModel(display, submitCh, cancelCh)
	model.configPath = opts.ConfigPath
	program := tea.NewProgram(model, tea.WithAltScreen())
	sink := &programSink{program: program}
	repl.SetProgressSink(sink)

	go runTurnHost(repl, program, submitCh, cancelCh)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "chat tui: %v\n", err)
		repl.CloseSession()
		return 1
	}
	repl.CloseSession()
	return 0
}

func runTurnHost(repl *chatrepl.Repl, program *tea.Program, submitCh <-chan string, cancelCh <-chan struct{}) {
	for text := range submitCh {
		done := make(chan TurnDoneMsg, 1)
		go func(t string) {
			result := repl.RunTurn(t)
			errText := ""
			if result.Failed {
				errText = result.Error
				if errText == "" {
					errText = "turn failed"
				}
			}
			done <- TurnDoneMsg{Reply: result.AssistantText, Err: errText}
		}(text)

		select {
		case msg := <-done:
			program.Send(msg)
		case <-cancelCh:
			repl.CancelTurn()
			program.Send(<-done)
		}
	}
}
