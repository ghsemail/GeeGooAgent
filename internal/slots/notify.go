package slots

import "github.com/ghsemail/GeeGooAgent/internal/tools"

// NotifyClarify emits a first-class clarify prompt so the Web option sheet
// can open after catalog tools have already finished.
func NotifyClarify(toolCtx tools.Context, question string, choices []string) {
	if toolCtx.Progress == nil {
		return
	}
	payload := map[string]any{
		"question": question,
		"choices":  append([]string(nil), choices...),
	}
	toolCtx.Progress("status", map[string]any{"phase": "clarify", "message": question})
	toolCtx.Progress("clarify", payload)
}
