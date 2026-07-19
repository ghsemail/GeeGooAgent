package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
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

func shouldHoldPlan(policy cognition.PlanPolicy, planGate bool, toolCtx tools.Context, mutating []llm.ToolCall) bool {
	if policy == nil {
		policy = cognition.DefaultPlanPolicy{}
	}
	return policy.ShouldHold(cognition.PlanHoldInput{
		GateEnabled:   planGate,
		Interactive:   toolCtx.Interactive,
		Approved:      toolCtx.Approved,
		MutatingCount: len(mutating),
	})
}

func planProposedPayload(policy cognition.PlanPolicy, mutating []llm.ToolCall) map[string]any {
	if policy == nil {
		policy = cognition.DefaultPlanPolicy{}
	}
	calls := make([]cognition.ProposedCall, 0, len(mutating))
	for _, call := range mutating {
		calls = append(calls, cognition.ProposedCall{Name: call.Name, Arguments: call.Arguments})
	}
	return policy.ProposedPayload(calls)
}

func planHoldUserMessage(policy cognition.PlanPolicy, planText string) string {
	if policy == nil {
		policy = cognition.DefaultPlanPolicy{}
	}
	return policy.HoldMessage(planText)
}
