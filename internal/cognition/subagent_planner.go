package cognition

// SubAgentPlanner skips LLM classify for delegated sub-turns. The parent already
// scoped the task; sub-agents should enter ReAct with stock gather tools directly.
type SubAgentPlanner struct{}

func (SubAgentPlanner) Plan(_ PlanInput) TurnPlan {
	plan := planForDomain(DomainStockAnalysis)
	plan.Mode = ModeGather
	plan.Act = "analyze"
	plan.Reason = "subagent: skip classify"
	plan.Confidence = 1
	return plan
}
