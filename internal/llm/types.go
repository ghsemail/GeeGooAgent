package llm

import (
	"context"
	"encoding/json"
)

// Role is a chat message role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one LLM conversation turn.
type Message struct {
	Role             Role       `json:"role"`
	Content          string     `json:"content,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
}

// ToolSchema describes a callable tool for the LLM.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall is a model-requested tool invocation.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// TokenUsage records token consumption.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	Model            string
}

// Response is a gateway result.
type Response struct {
	Content          string
	ToolCalls        []ToolCall
	Usage            TokenUsage
	ReasoningContent string
}

// Provider calls an LLM backend.
type Provider interface {
	Model() string
	Chat(ctx context.Context, messages []Message, tools []ToolSchema, temperature float64, maxTokens int) (*Response, error)
}

// ParseToolArguments decodes tool arguments from JSON string or map.
func ParseToolArguments(raw any) (map[string]any, error) {
	switch v := raw.(type) {
	case map[string]any:
		return v, nil
	case string:
		if v == "" {
			return map[string]any{}, nil
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(v), &out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return map[string]any{}, nil
	}
}
