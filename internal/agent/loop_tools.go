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
	return l.tools.ExecuteBatch(ctx, calls, toolCtx, step, func(event string, data map[string]any) {
		l.emit(event, data)
	})
}
