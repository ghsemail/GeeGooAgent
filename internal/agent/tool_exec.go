package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ToolProgress receives optional live tool events (chat UI / loop).
type ToolProgress func(event string, data map[string]any)

// ToolExec dispatches tool calls with timeout, optional approval, and batch parallelism.
// Shared by the ReAct loop and deterministic workflow runner.
type ToolExec struct {
	executor    *runtime.Executor
	timeout     time.Duration
	maxParallel int
	approvalFn  runtime.ApprovalFunc
}

// NewToolExec creates a tool dispatcher backed by the registry executor.
func NewToolExec(executor *runtime.Executor) *ToolExec {
	return &ToolExec{
		executor:    executor,
		timeout:     defaultToolTimeout,
		maxParallel: defaultToolMaxParallel,
	}
}

// SetMaxParallel caps concurrent tool executions in ExecuteBatch.
func (e *ToolExec) SetMaxParallel(n int) {
	if e == nil || n <= 0 {
		return
	}
	e.maxParallel = n
}

// SetTimeout bounds a single tool invocation.
func (e *ToolExec) SetTimeout(d time.Duration) {
	if e == nil || d <= 0 {
		return
	}
	e.timeout = d
}

// SetApproval wires interactive confirmation for mutating tools.
func (e *ToolExec) SetApproval(fn runtime.ApprovalFunc) {
	if e == nil {
		return
	}
	e.approvalFn = fn
}

// Execute runs one tool call with timeout and ctx cancellation.
func (e *ToolExec) Execute(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
	if e == nil || e.executor == nil {
		return tools.Result{Status: tools.StatusError, Summary: "tool executor not configured"}
	}
	if err := ctx.Err(); err != nil {
		return tools.Result{Status: tools.StatusError, Summary: fmt.Sprintf("已中断: %v", err)}
	}
	timedCtx, cancel := context.WithTimeout(ctx, tools.ExecutionTimeout(req.Name, e.timeout))
	defer cancel()
	tc := toolCtx
	tc.Ctx = timedCtx
	return e.run(req, tc)
}

// ExecuteBatch runs LLM tool_calls with approval, timeout, and optional parallelism.
func (e *ToolExec) ExecuteBatch(
	ctx context.Context,
	calls []llm.ToolCall,
	toolCtx tools.Context,
	step int,
	onProgress ToolProgress,
) []tools.Result {
	results := make([]tools.Result, len(calls))
	if len(calls) == 0 || e == nil {
		return results
	}

	runOne := func(i int, call llm.ToolCall) {
		if err := ctx.Err(); err != nil {
			results[i] = tools.Result{
				Status:  tools.StatusError,
				Summary: fmt.Sprintf("已中断: %v", err),
			}
			return
		}
		tc := toolCtx
		tc.Step = step
		if tc.Interactive && tools.ApprovalRequired(call.Name) && !tc.Approved {
			approved := false
			if e.approvalFn != nil {
				approved = e.approvalFn(call.Name, call.Arguments)
			}
			if !approved {
				result := tools.Result{
					Status:  tools.StatusSkip,
					Summary: "需要确认：" + call.Name + " 是写操作，请确认后再执行",
					Data:    map[string]any{"tool": call.Name, "approval_required": true},
				}
				emitProgress(onProgress, "tool_done", map[string]any{
					"step": step, "name": call.Name, "status": string(result.Status),
					"summary": result.Summary, "arguments": call.Arguments,
				})
				results[i] = result
				return
			}
			tc.Approved = true
		}
		emitProgress(onProgress, "tool_start", map[string]any{
			"step": step, "name": call.Name, "arguments": call.Arguments,
		})
		timedCtx, cancel := context.WithTimeout(ctx, tools.ExecutionTimeout(call.Name, e.timeout))
		defer cancel()
		tc.Ctx = timedCtx
		result := e.run(tools.CallRequest{Name: call.Name, Arguments: call.Arguments}, tc)
		if timedCtx.Err() != nil && result.Status == tools.StatusOK {
			result = tools.Result{
				Status:  tools.StatusError,
				Summary: fmt.Sprintf("工具超时或已中断: %v", timedCtx.Err()),
			}
		}
		emitProgress(onProgress, "tool_done", map[string]any{
			"step": step, "name": call.Name, "status": string(result.Status),
			"summary": result.Summary, "arguments": call.Arguments,
		})
		results[i] = result
	}

	if len(calls) == 1 || needsInteractiveApproval(toolCtx, calls) {
		for i, call := range calls {
			runOne(i, call)
		}
		return results
	}

	sem := make(chan struct{}, e.maxParallel)
	var wg sync.WaitGroup
	for i, call := range calls {
		if err := ctx.Err(); err != nil {
			for j := i; j < len(calls); j++ {
				results[j] = tools.Result{
					Status:  tools.StatusError,
					Summary: fmt.Sprintf("已中断: %v", err),
				}
			}
			return results
		}
		wg.Add(1)
		go func(i int, call llm.ToolCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			runOne(i, call)
		}(i, call)
	}
	wg.Wait()
	return results
}

func (e *ToolExec) run(req tools.CallRequest, toolCtx tools.Context) tools.Result {
	return e.executor.Execute(req, toolCtx)
}

func emitProgress(fn ToolProgress, event string, data map[string]any) {
	if fn != nil {
		fn(event, data)
	}
}

func needsInteractiveApproval(toolCtx tools.Context, calls []llm.ToolCall) bool {
	if !toolCtx.Interactive {
		return false
	}
	for _, call := range calls {
		if tools.ApprovalRequired(call.Name) {
			return true
		}
	}
	return false
}
