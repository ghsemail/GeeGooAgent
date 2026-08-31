package tools

import "time"

// ExecutionTimeout returns per-tool timeout, falling back to defaultTimeout.
func ExecutionTimeout(toolName string, defaultTimeout time.Duration) time.Duration {
	if override := matchPolicyRule(toolName); override == PolicyForbidden {
		return defaultTimeout
	}
	// Legacy map is folded into resolveToolMeta; lookup via zero registry inference.
	meta := resolveToolMeta(Tool{Name: toolName})
	if meta.Timeout > 0 {
		return meta.Timeout
	}
	return defaultTimeout
}

// ExecutionTimeoutFromRegistry prefers resolved metadata when registry is available.
func (r *Registry) ExecutionTimeout(toolName string, defaultTimeout time.Duration) time.Duration {
	if r != nil {
		if t, ok := r.tools[toolName]; ok && t.resolved.Timeout > 0 {
			return t.resolved.Timeout
		}
	}
	return ExecutionTimeout(toolName, defaultTimeout)
}
