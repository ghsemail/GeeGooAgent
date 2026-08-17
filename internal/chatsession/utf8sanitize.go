package chatsession

import (
	"strings"
	"unicode/utf8"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// ValidUTF8 returns s if it is valid UTF-8, otherwise replaces invalid sequences.
func ValidUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "\uFFFD")
}

// SanitizeForPersist normalizes all persisted text so PostgreSQL jsonb accepts the payload.
func (c *ChatSession) SanitizeForPersist() {
	if c == nil {
		return
	}
	c.Title = ValidUTF8(c.Title)
	c.Summary = ValidUTF8(c.Summary)
	c.Status = ValidUTF8(c.Status)
	for i := range c.Tags {
		c.Tags[i] = ValidUTF8(c.Tags[i])
	}
	for i := range c.ToolNames {
		c.ToolNames[i] = ValidUTF8(c.ToolNames[i])
	}
	c.Metadata = sanitizeMapUTF8(c.Metadata)
	for i := range c.Messages {
		sanitizeLLMMessage(&c.Messages[i])
	}
	for i := range c.StepRecords {
		c.StepRecords[i].Kind = ValidUTF8(c.StepRecords[i].Kind)
		c.StepRecords[i].ToolName = ValidUTF8(c.StepRecords[i].ToolName)
		c.StepRecords[i].ToolStatus = ValidUTF8(c.StepRecords[i].ToolStatus)
		c.StepRecords[i].Summary = ValidUTF8(c.StepRecords[i].Summary)
	}
}

func sanitizeLLMMessage(m *llm.Message) {
	if m == nil {
		return
	}
	m.Content = ValidUTF8(m.Content)
	m.ReasoningContent = ValidUTF8(m.ReasoningContent)
	m.ToolCallID = ValidUTF8(m.ToolCallID)
	for i := range m.ToolCalls {
		m.ToolCalls[i].ID = ValidUTF8(m.ToolCalls[i].ID)
		m.ToolCalls[i].Name = ValidUTF8(m.ToolCalls[i].Name)
		m.ToolCalls[i].Arguments = sanitizeMapUTF8(m.ToolCalls[i].Arguments)
	}
}

func sanitizeMapUTF8(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[ValidUTF8(k)] = sanitizeAnyUTF8(v)
	}
	return out
}

func sanitizeAnyUTF8(v any) any {
	switch x := v.(type) {
	case string:
		return ValidUTF8(x)
	case map[string]any:
		return sanitizeMapUTF8(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = sanitizeAnyUTF8(item)
		}
		return out
	default:
		return v
	}
}
