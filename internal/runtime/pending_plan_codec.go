package runtime

import (
	"encoding/json"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// PendingPlanToMap serializes a pending plan for session metadata storage.
func PendingPlanToMap(p *PendingPlan) map[string]any {
	if p == nil || len(p.ToolCalls) == 0 {
		return nil
	}
	calls := make([]any, 0, len(p.ToolCalls))
	for _, c := range p.ToolCalls {
		calls = append(calls, map[string]any{
			"id": c.ID, "name": c.Name, "arguments": c.Arguments,
		})
	}
	return map[string]any{"step": p.Step, "tool_calls": calls}
}

// PendingPlanFromMap restores a pending plan from session metadata.
func PendingPlanFromMap(raw any) *PendingPlan {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return nil
	}
	step := intFromAny(m["step"])
	items, ok := m["tool_calls"].([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	calls := make([]llm.ToolCall, 0, len(items))
	for _, item := range items {
		cm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		call := llm.ToolCall{
			ID:        stringFromAny(cm["id"]),
			Name:      stringFromAny(cm["name"]),
			Arguments: mapFromAny(cm["arguments"]),
		}
		if call.Name == "" {
			continue
		}
		calls = append(calls, call)
	}
	if len(calls) == 0 {
		return nil
	}
	return &PendingPlan{Step: step, ToolCalls: calls}
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func mapFromAny(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
