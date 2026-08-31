package tools

import (
	"encoding/json"
)

// RenderResultForLLM serializes a tool result for the LLM context window.
func RenderResultForLLM(toolName string, result Result, maxChars int) string {
	if maxChars <= 0 {
		maxChars = defaultMaxResultChars
	}
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
	text := string(raw)
	if len(text) > maxChars {
		return text[:maxChars]
	}
	_ = toolName // reserved for per-tool custom renderers
	return text
}

// RenderResult uses registry-resolved max chars when available.
func (r *Registry) RenderResult(toolName string, result Result) string {
	max := defaultMaxResultChars
	if r != nil {
		max = r.MaxResultChars(toolName)
	}
	return RenderResultForLLM(toolName, result, max)
}
