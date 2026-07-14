package runtime

// ProgressFunc receives live ReAct events for geegoo chat UI.
type ProgressFunc func(event string, data map[string]any)

// ApprovalFunc is invoked before executing a mutating tool in interactive chat.
// Return true to approve the call for this invocation.
type ApprovalFunc func(toolName string, args map[string]any) bool
