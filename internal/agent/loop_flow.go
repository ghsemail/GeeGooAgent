package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (l *Loop) tryChatWorkflow(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
) (runtime.TurnResult, bool) {
	if l == nil || session == nil {
		return runtime.TurnResult{}, false
	}
	composeLLM := l.effectiveSignalDiagnoseComposeLLM()
	composeSource, composeDetail := "", ""
	if l.resolveComposeBundle != nil {
		composeLLM, composeSource, composeDetail = l.resolveComposeBundle()
	}
	runner := &workflowchat.Runner{
		RunTool:              l.ExecuteTool,
		OnProgress:           l.onProgress,
		ComposeLLM:           composeLLM,
		ComposeModelSource:   composeSource,
		ComposeModelDetail:   composeDetail,
	}
	if l.strategyDevJev != nil {
		cfg := l.strategyDevJev()
		runner.JevConfig = &cfg
	}
	if l.signalDiagnoseLab != nil {
		cfg := l.signalDiagnoseLab()
		runner.LabConfig = &cfg
	}
	step := session.StepCounter + 1
	return runner.RunTurn(ctx, session, userText, toolCtx, step)
}
