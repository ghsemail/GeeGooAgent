package agent

// toolFirstPlaybooks are procedural skills that route directly to HTTP/MCP tools
// without needing long-term memory retrieval. Skipping the gate saves an auxiliary
// LLM round-trip and keeps tool payloads out of recall context.
var toolFirstPlaybooks = map[string]struct{}{
	"strategy-backtest":         {},
	"strategy-backtest-history": {},
	"strategy-backtest-run":     {},
	"strategy-signal-probe":     {},
}

// ShouldSkipRetrievalGate reports whether matched procedural skills are tool-first
// playbooks that do not benefit from long-term memory retrieval.
func ShouldSkipRetrievalGate(matchedSkills []string) bool {
	for _, name := range matchedSkills {
		if _, ok := toolFirstPlaybooks[name]; ok {
			return true
		}
	}
	return false
}
