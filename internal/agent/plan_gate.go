package agent

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func partitionToolCalls(calls []llm.ToolCall) (mutating, readonly []llm.ToolCall) {
	for _, call := range calls {
		if tools.ApprovalRequired(call.Name) {
			mutating = append(mutating, call)
		} else {
			readonly = append(readonly, call)
		}
	}
	return mutating, readonly
}

func shouldHoldPlan(planGate bool, toolCtx tools.Context, mutating []llm.ToolCall) bool {
	return planGate && toolCtx.Interactive && !toolCtx.Approved && len(mutating) > 0
}

func planProposedPayload(mutating []llm.ToolCall) map[string]any {
	toolsOut := make([]map[string]any, 0, len(mutating))
	names := make([]string, 0, len(mutating))
	for _, call := range mutating {
		names = append(names, call.Name)
		toolsOut = append(toolsOut, map[string]any{
			"name": call.Name, "arguments": call.Arguments,
		})
	}
	return map[string]any{"tools": names, "calls": toolsOut}
}

func planHoldUserMessage(planText string) string {
	planText = strings.TrimSpace(planText)
	if planText == "" {
		planText = "（模型未给出文字说明）"
	}
	return planText + "\n\n---\n写操作待确认：输入 y/yes/确认 执行，n/no/取消 放弃。"
}

func isPlanApproval(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "y", "yes", "确认", "执行", "ok":
		return true
	default:
		return false
	}
}

func isPlanRejection(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "n", "no", "取消", "skip":
		return true
	default:
		return false
	}
}
