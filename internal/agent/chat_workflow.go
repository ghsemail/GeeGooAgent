package agent

import (
	"context"
	"fmt"
	"strings"

	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

// RunChatWorkflowTurn runs one chat workflow step. handled is true when a workflow consumed the message.
func (l *Loop) RunChatWorkflowTurn(
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
		RunTool:            l.ExecuteTool,
		OnProgress:         l.onProgress,
		ComposeLLM:         composeLLM,
		ComposeModelSource: composeSource,
		ComposeModelDetail: composeDetail,
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

func activeWorkflowFragment(session *runtime.Session) ctxfrag.Fragment {
	if session == nil {
		return ctxfrag.StaticFragment{}
	}
	flow := workflowchat.LoadFromSession(session)
	if flow == nil || !flow.Active() {
		return ctxfrag.StaticFragment{}
	}
	var b strings.Builder
	b.WriteString("## Active chat workflow\n")
	fmt.Fprintf(&b, "- skill: %s\n- run_id: %s\n- phase: %s\n", flow.Template, flow.RunID, flow.Phase)
	b.WriteString("- Legacy path: run_chat_workflow_turn (only when chat_open_flow_tool is enabled). Prefer skill_view + ordinary tools for new sessions.\n")
	b.WriteString("- Pass through the workflow tool report as the user-facing answer when it contains the final summary.\n")
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 21}
}

var chatWorkflowSkillNames = map[string]struct{}{
	"signal_diagnose":             {},
	"strategy_dev":                {},
	"strategy_dev_pipeline":       {},
	"generate_strategy_archive":   {},
	"generate_strategy_cognition": {},
	"multi_strategy_compare":      {},
}

func needsChatWorkflowTool(matchedSkillNames []string, session *runtime.Session) bool {
	if session != nil {
		if flow := workflowchat.LoadFromSession(session); flow != nil && flow.Active() {
			return true
		}
	}
	// Keyword/catalog injection no longer auto-adds the open-flow tool for matched workflow skills.
	_ = matchedSkillNames
	return false
}

func chatWorkflowToolSchema() llm.ToolSchema {
	return llm.ToolSchema{
		Name: "run_chat_workflow_turn",
		Description: "推进或启动 Chat 工作流（信号诊断、策略开发、多策略对比、生成策略档案）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_message": map[string]any{
					"type":        "string",
					"description": "用户本回合原话或 继续 / 重试失败",
				},
			},
			"required": []any{"user_message"},
		},
	}
}
