package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const (
	defaultSubAgentMaxSteps = 20
	maxSubAgentMaxSteps     = 40
)

// SubAgent runs an isolated ReAct turn with its own step budget.
type SubAgent struct {
	gateway       *llm.Gateway
	executor      *runtime.Executor
	registry      *tools.Registry
	compressor    *prompt.Compressor
	eventBus      tools.EventEmitter
	maxSteps      int
	approvalFn    runtime.ApprovalFunc
	chatToolNames func() []string
}

// SubAgentConfig wires collaborators for delegated tasks.
type SubAgentConfig struct {
	Gateway       *llm.Gateway
	Executor      *runtime.Executor
	Registry      *tools.Registry
	MaxSteps      int
	ChatToolNames func() []string
}

// NewSubAgent creates a sub-agent runner.
func NewSubAgent(cfg SubAgentConfig) *SubAgent {
	maxSteps := cfg.MaxSteps
	if maxSteps <= 0 {
		maxSteps = defaultSubAgentMaxSteps
	}
	if maxSteps > maxSubAgentMaxSteps {
		maxSteps = maxSubAgentMaxSteps
	}
	return &SubAgent{
		gateway:       cfg.Gateway,
		executor:      cfg.Executor,
		registry:      cfg.Registry,
		maxSteps:      maxSteps,
		chatToolNames: cfg.ChatToolNames,
	}
}

// SetCompressor wires context compression for sub turns.
func (s *SubAgent) SetCompressor(c *prompt.Compressor) {
	if s != nil {
		s.compressor = c
	}
}

// SetEventBus wires observability for sub turns.
func (s *SubAgent) SetEventBus(bus tools.EventEmitter) {
	if s != nil {
		s.eventBus = bus
	}
}

// SetApproval mirrors parent interactive approval for mutating tools.
func (s *SubAgent) SetApproval(fn runtime.ApprovalFunc) {
	if s != nil {
		s.approvalFn = fn
	}
}

// Run executes one delegated task in an ephemeral session.
func (s *SubAgent) Run(parent tools.Context, task, background string, maxSteps int) tools.Result {
	if s == nil || s.gateway == nil || s.executor == nil || s.registry == nil {
		return tools.Result{Status: tools.StatusError, Summary: "delegate_task: sub-agent not configured", ExitCode: 1}
	}
	if parent.DelegateDepth >= 1 {
		return tools.Result{
			Status: tools.StatusError, Summary: "delegate_task: nested delegation is not allowed", ExitCode: 1,
		}
	}
	task = strings.TrimSpace(task)
	if task == "" {
		return tools.Result{Status: tools.StatusError, Summary: "delegate_task: task required", ExitCode: 1}
	}
	if parent.DryRun {
		return tools.Result{
			Status:  tools.StatusDryRun,
			Summary: fmt.Sprintf("dry-run: would delegate %q", truncateRunes(task, 80)),
			Data:    map[string]any{"task": task},
		}
	}
	if maxSteps <= 0 {
		maxSteps = s.maxSteps
	}
	if maxSteps > s.maxSteps {
		maxSteps = s.maxSteps
	}

	userText := task
	if bg := strings.TrimSpace(background); bg != "" {
		userText = "背景：\n" + bg + "\n\n任务：\n" + task
	}

	emit := parent.Progress
	if emit != nil {
		emit("subagent_start", map[string]any{"task": task, "max_steps": maxSteps})
	}

	session := runtime.NewSession()
	session.ID = parent.SessionID + ":sub"
	session.Messages[0] = llm.Message{Role: llm.RoleSystem, Content: chatprompt.SubAgentSystem()}

	loop := NewLoop(s.gateway, s.executor)
	loop.SetMaxToolRounds(maxSteps)
	loop.SetCompressor(s.compressor)
	loop.SetApproval(s.approvalFn)
	if s.eventBus != nil {
		loop.SetEventBus(s.eventBus)
	}
	if emit != nil {
		loop.SetProgress(func(event string, data map[string]any) {
			emit("subagent_event", map[string]any{"event": event, "data": data})
		})
	}

	childCtx := parent
	childCtx.DelegateDepth = parent.DelegateDepth + 1
	if childCtx.Ctx == nil {
		childCtx.Ctx = context.Background()
	}
	if emit != nil && childCtx.Progress == nil {
		childCtx.Progress = emit
	}

	schemas := subAgentSchemas(s.registry, s.chatToolNames, []string{"delegate_task"})
	result := loop.RunTurn(childCtx.GoContext(), session, userText, childCtx, schemas)

	if emit != nil {
		emit("subagent_end", map[string]any{
			"failed": result.Failed, "steps": len(result.StepRecords), "error": result.Error,
		})
	}
	if result.Failed {
		msg := result.AssistantText
		if msg == "" {
			msg = result.Error
		}
		return tools.Result{
			Status: tools.StatusError, Summary: "delegate_task failed: " + msg, ExitCode: 1,
			Data: map[string]any{"error": result.Error, "steps": len(result.StepRecords)},
		}
	}
	answer := strings.TrimSpace(result.AssistantText)
	summary := answer
	if len([]rune(summary)) > 300 {
		summary = truncateRunes(summary, 300) + "…"
	}
	return tools.Result{
		Status:  tools.StatusOK,
		Summary: "delegate_task: " + summary,
		Data: map[string]any{
			"answer": answer, "steps": len(result.StepRecords), "failed": false,
		},
	}
}

func subAgentSchemas(registry *tools.Registry, allowFn func() []string, exclude []string) []llm.ToolSchema {
	if registry == nil {
		return nil
	}
	allow := []string{}
	if allowFn != nil {
		allow = allowFn()
	}
	excluded := map[string]struct{}{}
	for _, name := range exclude {
		excluded[name] = struct{}{}
	}
	names := make([]string, 0, len(allow))
	for _, name := range allow {
		if _, skip := excluded[name]; skip {
			continue
		}
		if _, ok := registry.Get(name); ok {
			names = append(names, name)
		}
	}
	return registry.Schemas(names)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
