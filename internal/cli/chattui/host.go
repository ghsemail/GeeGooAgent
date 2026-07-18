package chattui

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// ReplHost adapts chatrepl.Repl for TUI slash commands and approval.
type ReplHost struct {
	Repl       *chatrepl.Repl
	ConfigPath string
	approveCh  chan bool
	askCh      chan approvalAsk

	clarifyAnsCh chan clarifyAnswer
	askClarifyCh chan clarifyAsk
}

type approvalAsk struct {
	Tool string
	Args string
}

// NewReplHost wraps a Repl for TUI use.
func NewReplHost(repl *chatrepl.Repl, configPath string) *ReplHost {
	h := &ReplHost{
		Repl:         repl,
		ConfigPath:   configPath,
		approveCh:    make(chan bool, 1),
		askCh:        make(chan approvalAsk, 1),
		clarifyAnsCh: make(chan clarifyAnswer, 1),
		askClarifyCh: make(chan clarifyAsk, 1),
	}
	repl.SetApprovalFn(h.promptApproval)
	repl.SetClarifyFn(h.promptClarify)
	return h
}

func (h *ReplHost) promptApproval(toolName string, args map[string]any) bool {
	summary := fmt.Sprintf("%v", args)
	if len(summary) > 200 {
		summary = summary[:197] + "..."
	}
	select {
	case h.askCh <- approvalAsk{Tool: toolName, Args: summary}:
	default:
	}
	return <-h.approveCh
}

func (h *ReplHost) promptClarify(question string, choices []string) (string, bool) {
	select {
	case h.askClarifyCh <- clarifyAsk{Question: question, Choices: append([]string(nil), choices...)}:
	default:
	}
	ans := <-h.clarifyAnsCh
	return ans.Answer, ans.OK
}

// PollApproval returns pending approval request if any.
func (h *ReplHost) PollApproval() (tool, args string, ok bool) {
	select {
	case a := <-h.askCh:
		return a.Tool, a.Args, true
	default:
		return "", "", false
	}
}

// AnswerApproval responds to the pending gate.
func (h *ReplHost) AnswerApproval(yes bool) {
	select {
	case h.approveCh <- yes:
	default:
	}
}

// PollClarify returns a pending clarify prompt if any.
func (h *ReplHost) PollClarify() (question string, choices []string, ok bool) {
	select {
	case a := <-h.askClarifyCh:
		return a.Question, a.Choices, true
	default:
		return "", nil, false
	}
}

// AnswerClarify responds to the pending clarify prompt.
func (h *ReplHost) AnswerClarify(answer string, ok bool) {
	select {
	case h.clarifyAnsCh <- clarifyAnswer{Answer: answer, OK: ok}:
	default:
	}
}

func (h *ReplHost) SessionInfo() string {
	r := h.Repl
	model := r.App.EffectiveLLMModel()
	return fmt.Sprintf("session=%s messages=%d dry_run=%v model=%s verbose=%v",
		r.Chat.ID, len(r.Chat.Messages), r.DryRun, model, r.Verbose)
}

func (h *ReplHost) SetVerbose(on bool) string {
	h.Repl.Verbose = on
	h.Repl.SetProgressSink(h.Repl.Progress) // re-attach
	if on {
		return "verbose=on（等同展开过程）"
	}
	// Map verbose off → collapsed details preference is caller's job
	return "verbose=off"
}

func (h *ReplHost) SetDryRun(on bool) string {
	h.Repl.DryRun = on
	return fmt.Sprintf("dry_run=%v", on)
}

func (h *ReplHost) ModelLine() string {
	if h.Repl == nil || h.Repl.App == nil {
		return ""
	}
	return h.Repl.App.EffectiveLLMModel()
}

// HandleSlash runs a REPL slash command and returns captured output for the TUI.
func (h *ReplHost) HandleSlash(line string) (quit bool, output string) {
	if h == nil || h.Repl == nil {
		return false, "无活动会话"
	}
	return h.Repl.HandleSlashCommand(line)
}

func (h *ReplHost) ThinkStatus() string {
	cfg := h.Repl.App.Config.LLM
	if cfg.Thinking == nil {
		return "auto"
	}
	if *cfg.Thinking {
		return "on"
	}
	return "off"
}

// ApplyVerboseToDisplay maps /verbose onto details_mode.
func ApplyVerboseToDisplay(d *config.DisplayConfig, on bool) {
	if d == nil {
		return
	}
	d.Normalize()
	if on {
		d.DetailsMode = config.ModeExpanded
		d.Sections.Thinking = config.ModeExpanded
		d.Sections.Tools = config.ModeExpanded
	} else {
		d.DetailsMode = config.ModeCollapsed
		d.Sections.Thinking = ""
		d.Sections.Tools = ""
	}
	d.Normalize()
}

func parseOnOff(args []string) (bool, bool) {
	if len(args) == 0 {
		return false, false
	}
	switch strings.ToLower(args[0]) {
	case "on", "1", "true":
		return true, true
	case "off", "0", "false":
		return false, true
	default:
		return false, false
	}
}