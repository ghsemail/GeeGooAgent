package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
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
func ShouldSkipRetrievalGate(matchedSkills []string, plan cognition.TurnPlan, userText string) bool {
	if plan.Mode == cognition.ModeClarify {
		return true
	}
	for _, name := range matchedSkills {
		if _, ok := toolFirstPlaybooks[name]; ok {
			return true
		}
	}
	return !retrievalgate.HasMemoryCue(userText)
}

func skipRetrievalReason(matchedSkills []string, plan cognition.TurnPlan, userText string) string {
	if plan.Mode == cognition.ModeClarify {
		return "clarify turn"
	}
	for _, name := range matchedSkills {
		if _, ok := toolFirstPlaybooks[name]; ok {
			return "tool-first playbook"
		}
	}
	if !retrievalgate.HasMemoryCue(userText) {
		return "no memory cue"
	}
	return "skip"
}
