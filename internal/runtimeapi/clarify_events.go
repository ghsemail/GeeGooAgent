package runtimeapi

func clarifyResolvedPayload(p PendingClarify, answer string, auto bool) map[string]any {
	return map[string]any{
		"session_id": p.SessionID,
		"question":   p.Question,
		"choices":    p.Choices,
		"answer":     answer,
		"auto":       auto,
	}
}

func emitClarifyResolved(emit func(string, map[string]any), p PendingClarify, answer string, auto bool) {
	payload := clarifyResolvedPayload(p, answer, auto)
	emit("clarify_resolved", payload)
	if auto {
		emit("clarify_auto_resolved", payload)
	}
}

func newChatStreamClarifyHooks(emit func(string, map[string]any)) ClarifyHooks {
	return ClarifyHooks{
		OnPending: func(p PendingClarify) {
			emit("clarify", map[string]any{
				"session_id": p.SessionID,
				"question":   p.Question,
				"choices":    p.Choices,
			})
		},
		OnResolved: func(p PendingClarify, answer string, auto bool) {
			emitClarifyResolved(emit, p, answer, auto)
		},
	}
}

func newAgentEventClarifyHooks(write func(map[string]any)) ClarifyHooks {
	return ClarifyHooks{
		OnPending: func(p PendingClarify) {
			write(map[string]any{
				"event":      "clarify",
				"session_id": p.SessionID,
				"question":   p.Question,
				"choices":    p.Choices,
			})
		},
		OnResolved: func(p PendingClarify, answer string, auto bool) {
			payload := clarifyResolvedPayload(p, answer, auto)
			payload["event"] = "clarify_resolved"
			write(payload)
			if auto {
				autoPayload := cloneStringAnyMap(payload)
				autoPayload["event"] = "clarify_auto_resolved"
				write(autoPayload)
			}
		},
	}
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
