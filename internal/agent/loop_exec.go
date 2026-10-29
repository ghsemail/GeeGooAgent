package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ExecuteTool runs one tool through the shared dispatcher.
func (l *Loop) ExecuteTool(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
	if l == nil || l.tools == nil {
		return tools.Result{Status: tools.StatusError, Summary: "tool executor not configured"}
	}
	return l.tools.Execute(ctx, req, toolCtx)
}
