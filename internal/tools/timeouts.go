package tools

import "time"

// Long-running tools need more than the default 120s agent tool timeout.
var toolExecutionTimeouts = map[string]time.Duration{
	"generate_dca_strategy":  7 * 60 * time.Second, // 7 sequential LLM calls on analyze-api
	"generate_grid_strategy": 5 * 60 * time.Second, // 3 LLM calls
	"get_mcp_analysis":       3 * time.Minute,
}

// ExecutionTimeout returns per-tool timeout, falling back to defaultTimeout.
func ExecutionTimeout(toolName string, defaultTimeout time.Duration) time.Duration {
	if d, ok := toolExecutionTimeouts[toolName]; ok {
		return d
	}
	return defaultTimeout
}
