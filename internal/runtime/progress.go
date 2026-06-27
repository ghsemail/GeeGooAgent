package runtime

// ProgressFunc receives live ReAct events for geegoo chat UI.
type ProgressFunc func(event string, data map[string]any)
