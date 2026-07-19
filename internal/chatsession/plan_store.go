package chatsession

import "github.com/ghsemail/GeeGooAgent/internal/llm"

// SyncHeldPlan persists or clears a held mutating-tool plan in Metadata.
func (c *ChatSession) SyncHeldPlan(step int, calls []llm.ToolCall) {
	if c == nil {
		return
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}
	if len(calls) == 0 {
		delete(c.Metadata, "pending_plan")
		return
	}
	items := make([]any, 0, len(calls))
	for _, call := range calls {
		items = append(items, map[string]any{
			"id": call.ID, "name": call.Name, "arguments": call.Arguments,
		})
	}
	c.Metadata["pending_plan"] = map[string]any{"step": step, "tool_calls": items}
}

// HeldPlanFromMetadata reads a held plan from Metadata.
func (c *ChatSession) HeldPlanFromMetadata() (step int, calls []llm.ToolCall, ok bool) {
	if c == nil || c.Metadata == nil {
		return 0, nil, false
	}
	raw, exists := c.Metadata["pending_plan"]
	if !exists {
		return 0, nil, false
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return 0, nil, false
	}
	step = intMeta(m, "step")
	items, ok := m["tool_calls"].([]any)
	if !ok || len(items) == 0 {
		return 0, nil, false
	}
	calls = make([]llm.ToolCall, 0, len(items))
	for _, item := range items {
		cm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strMeta(cm, "name")
		if name == "" {
			continue
		}
		calls = append(calls, llm.ToolCall{
			ID: strMeta(cm, "id"), Name: name, Arguments: argsMeta(cm, "arguments"),
		})
	}
	if len(calls) == 0 {
		return 0, nil, false
	}
	return step, calls, true
}

func argsMeta(m map[string]any, key string) map[string]any {
	raw, ok := m[key].(map[string]any)
	if !ok || raw == nil {
		return map[string]any{}
	}
	return raw
}
