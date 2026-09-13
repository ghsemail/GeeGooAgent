package cognition

const (
	// RoutingModeAgentContext: TurnPlan is observability-only; ReAct agent routes tools.
	RoutingModeAgentContext = "agent_context"
	// RoutingModeLegacy: TurnPlan drives tool filter, preset clarify, sticky guards.
	RoutingModeLegacy = "legacy"
)

// NormalizeRoutingMode returns agent_context when empty or unknown (default).
func NormalizeRoutingMode(mode string) string {
	switch mode {
	case RoutingModeLegacy:
		return RoutingModeLegacy
	default:
		return RoutingModeAgentContext
	}
}

// AgentContextRouting reports observability-only routing (no TurnPlan-driven bypass).
func AgentContextRouting(mode string) bool {
	return NormalizeRoutingMode(mode) == RoutingModeAgentContext
}
