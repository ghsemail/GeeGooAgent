package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const (
	defaultMaxToolRounds = 80
	defaultToolMaxParallel = 4
	defaultToolTimeout     = 120 * time.Second
)

// Loop runs plan → act → observe for one chat turn.
type Loop struct {
	gateway       *llm.Gateway
	tools         *ToolExec
	maxToolRounds int
	onProgress    runtime.ProgressFunc
	compressor    *prompt.Compressor
	eventBus      tools.EventEmitter
}

// NewLoop creates an agent loop.
func NewLoop(gateway *llm.Gateway, executor *runtime.Executor) *Loop {
	return &Loop{
		gateway:       gateway,
		tools:         NewToolExec(executor),
		maxToolRounds: defaultMaxToolRounds,
	}
}

// ToolExec returns the shared tool dispatcher (also used by workflow).
func (l *Loop) ToolExec() *ToolExec {
	if l == nil {
		return nil
	}
	return l.tools
}

// SetMaxToolRounds sets the per-turn LLM↔tool iteration cap (config max_steps).
func (l *Loop) SetMaxToolRounds(n int) {
	if l == nil {
		return
	}
	if n <= 0 {
		n = defaultMaxToolRounds
	}
	if n > 90 {
		n = 90
	}
	l.maxToolRounds = n
}

// SetToolMaxParallel caps concurrent tool executions per LLM round.
func (l *Loop) SetToolMaxParallel(n int) {
	if l != nil && l.tools != nil {
		l.tools.SetMaxParallel(n)
	}
}

// SetToolTimeout bounds a single tool invocation.
func (l *Loop) SetToolTimeout(d time.Duration) {
	if l != nil && l.tools != nil {
		l.tools.SetTimeout(d)
	}
}

// SetGateway swaps the LLM gateway (e.g. after /think or /model).
func (l *Loop) SetGateway(gateway *llm.Gateway) {
	l.gateway = gateway
}

// SetCompressor wires optional context compaction before LLM calls.
func (l *Loop) SetCompressor(c *prompt.Compressor) {
	l.compressor = c
}

// SetProgress wires live step output (geegoo chat verbose UI).
func (l *Loop) SetProgress(fn runtime.ProgressFunc) {
	l.onProgress = fn
}

// SetApproval wires interactive confirmation for mutating tools.
func (l *Loop) SetApproval(fn runtime.ApprovalFunc) {
	if l != nil && l.tools != nil {
		l.tools.SetApproval(fn)
	}
}

// SetEventBus wires turn- and tool-level observability (ToolCalled is also emitted by the registry).
func (l *Loop) SetEventBus(bus tools.EventEmitter) {
	l.eventBus = bus
}

func (l *Loop) emit(event string, data map[string]any) {
	if l.onProgress != nil {
		l.onProgress(event, data)
	}
}

func (l *Loop) emitBus(event string, data map[string]any) {
	if l.eventBus != nil {
		l.eventBus.Emit(event, data)
	}
}

func (l *Loop) RunTurn(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
) runtime.TurnResult {
	if ctx == nil {
		ctx = context.Background()
	}
	toolCtx.Ctx = ctx
	if toolCtx.EventBus == nil && l.eventBus != nil {
		toolCtx.EventBus = l.eventBus
	}
	if toolCtx.Interactive {
		schemas = filterInteractiveSchemas(schemas)
	}

	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: userText})
	messages := session.LLMMessages()
	records := []runtime.StepRecord{}

	l.emit("turn_start", map[string]any{"user_text": userText})
	l.emitBus("TurnStarted", map[string]any{
		"session_id": session.ID, "user_text": userText,
	})
	messages = l.applyHygiene(ctx, session, messages)

	for round := 0; round < l.maxToolRounds; round++ {
		if err := ctx.Err(); err != nil {
			return l.failTurn(ctx, session, err, records)
		}
		done, result := l.runRound(ctx, session, &messages, toolCtx, schemas, round, &records)
		if done {
			if !result.Failed {
				l.emitBus("TurnCompleted", map[string]any{
					"session_id": session.ID, "steps": len(result.StepRecords),
				})
			}
			return result
		}
	}

	msg := l.finishBudgetExhausted(ctx, session, messages, records)
	return msg
}

func (l *Loop) failTurn(ctx context.Context, session *runtime.Session, err error, records []runtime.StepRecord) runtime.TurnResult {
	msg := fmt.Sprintf("已中断: %v", err)
	l.emit("error", map[string]any{"message": msg})
	l.emitBus("TurnFailed", map[string]any{
		"session_id": session.ID, "error": err.Error(),
	})
	return runtime.TurnResult{AssistantText: msg, Failed: true, Error: err.Error(), StepRecords: records}
}
