package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

// toolFirstPlaybooks are procedural skills that route directly to HTTP/MCP tools
// without needing long-term memory retrieval. Skipping the gate saves an auxiliary
// LLM round-trip and keeps tool payloads out of recall context.
var toolFirstPlaybooks = map[string]struct{}{
	"strategy-backtest":         {},
	"strategy-backtest-history": {},
	"strategy-backtest-run":     {},
	"strategy-signal-probe":     {},
}

// ShouldSkipRetrievalGate reports whether the turn can skip the LLM retrieval gate.
// Clarify turns and tool-first playbooks do not need long-term memory recall.
func ShouldSkipRetrievalGate(matchedSkills []string, plan cognition.TurnPlan) bool {
	if plan.Mode == cognition.ModeClarify {
		return true
	}
	for _, name := range matchedSkills {
		if _, ok := toolFirstPlaybooks[name]; ok {
			return true
		}
	}
	return false
}
