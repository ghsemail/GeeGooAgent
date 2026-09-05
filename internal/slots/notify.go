package slots

import "github.com/ghsemail/GeeGooAgent/internal/tools"

// NotifyClarify emits a first-class clarify prompt on the chat progress
// stream so the Web option sheet can open while a catalog tool is still open.
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

func emitCatalogToolDone(toolCtx tools.Context, name string, res tools.Result, args map[string]any) {
	if toolCtx.Progress == nil {
		return
	}
	toolCtx.Progress("tool_done", map[string]any{
		"name":      name,
		"status":    string(res.Status),
		"summary":   res.Summary,
		"arguments": args,
	})
}
