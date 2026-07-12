package progress

// Sink receives ReAct / Agent live progress events for CLI or TUI rendering.
type Sink interface {
	EmitProgress(event string, data map[string]any)
}
