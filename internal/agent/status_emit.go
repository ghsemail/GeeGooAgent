package agent

// emitStatus sends a lightweight human-readable progress line for chat SSE / harness UI.
func (l *Loop) emitStatus(phase, message string) {
	if message == "" {
		return
	}
	payload := map[string]any{"phase": phase, "message": message}
	if phase != "" {
		payload["phase"] = phase
	}
	l.emit("status", payload)
}
