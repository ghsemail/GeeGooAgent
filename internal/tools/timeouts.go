package tools

import "time"

// Long-running tools need more than the default 120s agent tool timeout.
var toolExecutionTimeouts = map[string]time.Duration{
	"generate_dca_strategy":  4 * 60 * time.Second, // 2 analysis + 1 batch translation on analyze-api
	"generate_grid_strategy": 3 * 60 * time.Second, // 1 analysis + optional batch translation
	"get_mcp_analysis":       3 * time.Minute,
}

// ExecutionTimeout returns per-tool timeout, falling back to defaultTimeout.
func ExecutionTimeout(toolName string, defaultTimeout time.Duration) time.Duration {
	if d, ok := toolExecutionTimeouts[toolName]; ok {
		return d
	}
	return defaultTimeout
}
