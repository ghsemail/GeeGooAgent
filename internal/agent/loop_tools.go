package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (l *Loop) executeToolCalls(
	ctx context.Context,
	calls []llm.ToolCall,
	toolCtx tools.Context,
	step int,
) []tools.Result {
	results := make([]tools.Result, len(calls))
	pending := make([]llm.ToolCall, 0, len(calls))
	pendingIdx := make([]int, 0, len(calls))

	for i, call := range calls {
		if res, ok := l.interceptToolCall(call, toolCtx); ok {
			results[i] = res
			l.emit("tool_intercepted", map[string]any{
				"step": step, "name": call.Name, "status": string(res.Status), "summary": res.Summary,
			})
			l.emit("tool_done", map[string]any{
				"step": step, "name": call.Name, "status": string(res.Status),
				"summary": res.Summary, "arguments": call.Arguments,
			})
			continue
		}
		pendingIdx = append(pendingIdx, i)
		pending = append(pending, call)
	}
	if len(pending) == 0 {
		return results
	}

	batch := l.tools.ExecuteBatch(ctx, pending, toolCtx, step, func(event string, data map[string]any) {
		l.emit(event, data)
	})
	for j, idx := range pendingIdx {
		results[idx] = batch[j]
	}
	return results
}
