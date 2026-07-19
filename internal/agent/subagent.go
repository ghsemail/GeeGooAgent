package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
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
	mem           memport.Port
	eventBus      tools.EventEmitter
	maxSteps      int
	maxParallel   int
	approvalFn    runtime.ApprovalFunc
	chatToolNames func() []string
}

// SubAgentConfig wires collaborators for delegated tasks.
type SubAgentConfig struct {
	Gateway       *llm.Gateway
	Executor      *runtime.Executor
	Registry      *tools.Registry
	MaxSteps      int
	MaxParallel   int
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
	par := cfg.MaxParallel
	if par <= 0 {
		par = 3
	}
	if par > 8 {
		par = 8
	}
	return &SubAgent{
		gateway:       cfg.Gateway,
		executor:      cfg.Executor,
		registry:      cfg.Registry,
		maxSteps:      maxSteps,
		maxParallel:   par,
		chatToolNames: cfg.ChatToolNames,
	}
}

// SetCompressor wires context compression for sub turns (Memory port).
func (s *SubAgent) SetCompressor(c *prompt.Compressor) {
	if s != nil {
		s.compressor = c
		s.SetMemory(memory.NewAdapter(memory.AdapterConfig{Compressor: c}))
	}
}

// SetMemory wires the Memory port for sub turns.
func (s *SubAgent) SetMemory(m memport.Port) {
	if s != nil {
		if m != nil {
			s.mem = m
		} else {
			s.mem = memport.Noop()
		}
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

// DelegateTask implements tools.TaskDelegator.
func (s *SubAgent) DelegateTask(ctx tools.Context, task, background string, maxSteps int) tools.Result {
	return s.Run(ctx, task, background, maxSteps)
}

// DelegateTasks runs multiple sub-agent tasks with bounded parallelism.
func (s *SubAgent) DelegateTasks(parent tools.Context, specs []tools.BatchDelegateTask) tools.Result {
	if parent.DelegateDepth >= 1 {
		return tools.Result{
			Status: tools.StatusError, Summary: "delegate_tasks: nested delegation is not allowed", ExitCode: 1,
		}
	}
	if len(specs) == 0 {
		return tools.Result{Status: tools.StatusError, Summary: "delegate_tasks: tasks required", ExitCode: 1}
	}
	if len(specs) > 8 {
		return tools.Result{Status: tools.StatusError, Summary: "delegate_tasks: at most 8 tasks per call", ExitCode: 1}
	}
	if parent.DryRun {
		return tools.Result{
			Status:  tools.StatusDryRun,
			Summary: fmt.Sprintf("dry-run: would delegate %d task(s)", len(specs)),
			Data:    map[string]any{"count": len(specs)},
		}
	}
	type item struct {
		idx int
		res tools.Result
	}
	out := make([]map[string]any, len(specs))
	sem := make(chan struct{}, s.maxParallel)
	ch := make(chan item, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		wg.Add(1)
		go func(i int, spec tools.BatchDelegateTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := s.Run(parent, spec.Task, spec.Background, spec.MaxSteps)
			ch <- item{idx: i, res: res}
		}(i, spec)
	}
	wg.Wait()
	close(ch)
	ok, fail := 0, 0
	for it := range ch {
		res := it.res
		answer, _ := res.Data["answer"].(string)
		out[it.idx] = map[string]any{
			"task": specs[it.idx].Task, "status": string(res.Status),
			"summary": res.Summary, "answer": answer,
		}
		if res.Status == tools.StatusOK {
			ok++
		} else {
			fail++
		}
	}
	status := tools.StatusOK
	summary := fmt.Sprintf("delegate_tasks: %d ok, %d failed", ok, fail)
	if fail > 0 && ok == 0 {
		status = tools.StatusError
	}
	return tools.Result{
		Status: status, Summary: summary,
		Data: map[string]any{"results": out, "ok": ok, "failed": fail},
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
	if s.mem != nil {
		loop.SetMemory(s.mem)
	} else {
		loop.SetCompressor(s.compressor)
	}
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

	schemas := subAgentSchemas(s.registry, s.chatToolNames, []string{"delegate_task", "delegate_tasks"})
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
