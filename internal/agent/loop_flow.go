package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/taskflow"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (l *Loop) tryTaskFlow(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
) (runtime.TurnResult, bool) {
	if l == nil || session == nil {
		return runtime.TurnResult{}, false
	}
	runner := &taskflow.Runner{
		RunTool:    l.ExecuteTool,
		OnProgress: l.onProgress,
	}
	step := session.StepCounter + 1
	return runner.RunTurn(ctx, session, userText, toolCtx, step)
}
