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
