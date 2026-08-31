package tools

import "strings"

// ApprovalRequired reports whether a tool performs a mutating/dangerous
// operation that should be confirmed before execution in interactive chat.
func ApprovalRequired(toolName string) bool {
	return EffectivePolicyForName(toolName) == PolicyPrompt
}

// EffectivePolicyForName resolves policy without a registry (inference + YAML rules).
func EffectivePolicyForName(toolName string) PolicyAction {
	policy := inferPolicy(toolName)
	if override := matchPolicyRule(toolName); override != "" {
		return override
	}
	return policy
}

// ApprovalGate wraps a tool handler with a confirmation check.
func ApprovalGate(name string, handle Handler) Handler {
	if !ApprovalRequired(name) {
		return handle
	}
	return func(ctx Context, args map[string]any) Result {
		if ctx.DryRun {
			return handle(ctx, args)
		}
		if ctx.Approved || !ctx.Interactive {
			return handle(ctx, args)
		}
		return Result{
			Status:  StatusSkip,
			Summary: "需要确认：" + name + " 是写操作，请确认后再执行",
			Data:    map[string]any{"tool": name, "approval_required": true},
		}
	}
}

// IsMutatingPrefix is a legacy helper for tests; prefer ApprovalRequired.
func IsMutatingPrefix(toolName string) bool {
	name := strings.ToLower(toolName)
	for _, prefix := range []string{"create_", "update_", "delete_", "edit_", "switch_", "add_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
