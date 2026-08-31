package tools

import (
	"encoding/json"
	"strings"
)

// RenderResultForLLM serializes a tool result for the LLM context window.
func RenderResultForLLM(toolName string, result Result, maxChars int) string {
	if maxChars <= 0 {
		maxChars = defaultMaxResultChars
	}
	text := defaultRenderJSON(toolName, result)
	if len(text) > maxChars {
		return text[:maxChars]
	}
	return text
}

func defaultRenderJSON(toolName string, result Result) string {
	type payload struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
		Data    any    `json:"data,omitempty"`
	}
	raw, _ := json.Marshal(payload{
		Status:  string(result.Status),
		Summary: result.Summary,
		Data:    result.Data,
	})
	_ = toolName
	return string(raw)
}

// RenderListSummary produces a compact summary for list_* tools (Tier1).
func RenderListSummary(toolName string, result Result, maxChars int) string {
	if maxChars <= 0 {
		maxChars = defaultMaxResultChars
	}
	items, ok := result.Data["items"].([]any)
	if !ok {
		if itemsRaw, ok := result.Data["items"]; ok {
			switch v := itemsRaw.(type) {
			case []map[string]any:
				items = make([]any, len(v))
				for i, row := range v {
					items[i] = row
				}
			}
		}
	}
	if len(items) == 0 {
		return RenderResultForLLM(toolName, result, maxChars)
	}
	preview := make([]any, 0, 5)
	for i, item := range items {
		if i >= 5 {
			break
		}
		preview = append(preview, item)
	}
	summary := map[string]any{
		"status":  string(result.Status),
		"summary": result.Summary,
		"count":   len(items),
		"preview": preview,
	}
	if len(items) > 5 {
		summary["truncated"] = true
	}
	raw, _ := json.Marshal(summary)
	text := string(raw)
	if len(text) > maxChars {
		return text[:maxChars]
	}
	return text
}

// RenderResult uses registry-resolved renderer and max chars when available.
func (r *Registry) RenderResult(toolName string, result Result) string {
	max := defaultMaxResultChars
	var render RenderFunc
	if r != nil {
		max = r.MaxResultChars(toolName)
		if t, ok := r.tools[toolName]; ok {
			render = t.RenderForLLM
			if render == nil && strings.HasPrefix(toolName, "list_") {
				render = RenderListSummary
			}
		}
	}
	if render != nil {
		return render(toolName, result, max)
	}
	return RenderResultForLLM(toolName, result, max)
}
