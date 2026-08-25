package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/playbookexec"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// SetPlaybookRouter wires deterministic playbook executors into the chat loop.
func (l *Loop) SetPlaybookRouter(r *playbookexec.Router) {
	if l == nil {
		return
	}
	l.playbookRouter = r
}

// ExecuteTool runs one tool through the shared dispatcher.
func (l *Loop) ExecuteTool(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
	if l == nil || l.tools == nil {
		return tools.Result{Status: tools.StatusError, Summary: "tool executor not configured"}
	}
	return l.tools.Execute(ctx, req, toolCtx)
}
