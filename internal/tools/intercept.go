package tools

// IsWorkflowExclusiveTool reports whether name belongs to report_workflow-only tools
// (not shared with market/strategy/etc.). Interactive chat should not invoke these.
func IsWorkflowExclusiveTool(name string) bool {
	_, ok := workflowExclusiveTools[name]
	return ok
}

// WorkflowExclusiveToolNames returns sorted workflow-only tool names.
func WorkflowExclusiveToolNames() []string {
	out := make([]string, 0, len(workflowExclusiveTools))
	for name := range workflowExclusiveTools {
		out = append(out, name)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
