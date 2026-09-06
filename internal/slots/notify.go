package slots

import "github.com/ghsemail/GeeGooAgent/internal/tools"

// NotifyClarify is a no-op. The option sheet must open only after ClarifyHub
// has a waiter; chat/stream emits SSE from Wait's onPending callback.
func NotifyClarify(toolCtx tools.Context, question string, choices []string) {
	_ = toolCtx
	_ = question
	_ = choices
}
