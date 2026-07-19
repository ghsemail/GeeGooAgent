package cognition

import "strings"

// DefaultPlanPolicy matches the historical agent plan_gate behavior.
type DefaultPlanPolicy struct{}

// ShouldHold reports whether mutating tools must wait for explicit confirmation.
func (DefaultPlanPolicy) ShouldHold(in PlanHoldInput) bool {
	return in.GateEnabled && in.Interactive && !in.Approved && in.MutatingCount > 0
}

// IsApproval reports user text that confirms a pending plan.
func (DefaultPlanPolicy) IsApproval(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "y", "yes", "确认", "执行", "ok":
		return true
	default:
		return false
	}
}

// IsRejection reports user text that cancels a pending plan.
func (DefaultPlanPolicy) IsRejection(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "n", "no", "取消", "skip":
		return true
	default:
		return false
	}
}

// HoldMessage builds the user-facing hold prompt.
func (DefaultPlanPolicy) HoldMessage(planText string) string {
	planText = strings.TrimSpace(planText)
	if planText == "" {
		planText = "（模型未给出文字说明）"
	}
	return planText + "\n\n---\n写操作待确认：输入 y/yes/确认 执行，n/no/取消 放弃。"
}

// ProposedPayload builds the plan_proposed event payload.
func (DefaultPlanPolicy) ProposedPayload(calls []ProposedCall) map[string]any {
	toolsOut := make([]map[string]any, 0, len(calls))
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		names = append(names, call.Name)
		toolsOut = append(toolsOut, map[string]any{
			"name": call.Name, "arguments": call.Arguments,
		})
	}
	return map[string]any{"tools": names, "calls": toolsOut}
}
