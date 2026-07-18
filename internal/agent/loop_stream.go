package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func (l *Loop) streamHandler(ctx context.Context) llm.StreamHandler {
	outer := llm.StreamHandlerFrom(ctx)
	if outer == nil && l.onProgress == nil {
		return nil
	}
	var thinking bool
	toolGen := map[int]bool{}
	return func(delta llm.StreamDelta) {
		if delta.ReasoningContent != "" {
			if !thinking {
				thinking = true
				l.emit("thinking_start", map[string]any{})
			}
			l.emit("stream_delta", map[string]any{"reasoning": delta.ReasoningContent})
		}
		if delta.Content != "" {
			if thinking {
				thinking = false
				l.emit("thinking_stop", map[string]any{})
			}
			l.emit("stream_delta", map[string]any{"content": delta.Content})
		}
		if delta.ToolCall != nil {
			if thinking {
				thinking = false
				l.emit("thinking_stop", map[string]any{})
			}
			tc := delta.ToolCall
			if !toolGen[tc.Index] {
				toolGen[tc.Index] = true
				l.emit("tool_gen_start", map[string]any{
					"index": tc.Index, "id": tc.ID, "name": tc.Name,
				})
			} else if tc.Name != "" {
				l.emit("tool_gen_delta", map[string]any{
					"index": tc.Index, "name": tc.Name,
				})
			}
			if tc.Arguments != "" {
				l.emit("tool_gen_delta", map[string]any{
					"index": tc.Index, "arguments": tc.Arguments,
				})
			}
		}
		if outer != nil {
			outer(delta)
		}
	}
}

func (l *Loop) emitStepComplete(step, round int, hadTools bool, toolNames []string) {
	l.emit("step_complete", map[string]any{
		"step": step, "round": round + 1,
		"had_tools": hadTools, "tool_names": toolNames,
	})
}
