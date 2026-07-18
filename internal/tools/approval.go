package tools

import "strings"

// ApprovalRequired reports whether a tool performs a mutating/dangerous
// operation that should be confirmed before execution in interactive chat.
// Workflow (pre_market) runs are deterministic and pre-authorized, so this
// gate only applies to ad-hoc chat invocations.
//
// The check is name-based and conservative: any create_/update_/delete_/edit_/switch
// tool is considered mutating. Read-only tools (list_/get_/search_) are not.
var ApprovalRequired = func(toolName string) bool {
	name := strings.ToLower(toolName)
	for _, prefix := range []string{"create_", "update_", "delete_", "edit_", "switch_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// ApprovalGate wraps a tool handler with a confirmation check. When the
// context marks the session as interactive (Interactive=true) and the tool
// is mutating, the handler is skipped with StatusSkip unless ctx.Approved
// is true. Workflow callers set Approved=true; chat sets Interactive=true
// and prompts the user via the UI before setting Approved.
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
