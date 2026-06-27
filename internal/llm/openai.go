package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider calls OpenAI-compatible chat/completions endpoints.
type OpenAIProvider struct {
	model           string
	apiKey          string
	baseURL         string
	httpClient      *http.Client
	thinkingEnabled bool
	reasoningEffort string
}

// OpenAIOptions configures the HTTP provider.
type OpenAIOptions struct {
	Model           string
	APIKey          string
	BaseURL         string
	HTTPClient      *http.Client
	ThinkingEnabled bool
	ReasoningEffort string
}

// NewOpenAIProvider creates an OpenAI-compatible provider.
func NewOpenAIProvider(opts OpenAIOptions) *OpenAIProvider {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		model:           opts.Model,
		apiKey:          opts.APIKey,
		baseURL:         base,
		httpClient:      client,
		thinkingEnabled: opts.ThinkingEnabled,
		reasoningEffort: opts.ReasoningEffort,
	}
}

func (p *OpenAIProvider) Model() string { return p.model }

func (p *OpenAIProvider) Chat(messages []Message, tools []ToolSchema, temperature float64, maxTokens int) (*Response, error) {
	body := map[string]any{
		"model":       p.model,
		"messages":    toOpenAIMessages(messages),
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}
	if len(tools) > 0 {
		body["tools"] = toOpenAITools(tools)
	}
	if p.thinkingEnabled {
		body["thinking"] = map[string]any{"type": "enabled"}
		if effort := strings.TrimSpace(p.reasoningEffort); effort != "" {
			body["reasoning_effort"] = effort
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	return parseOpenAIResponse(respBody, p.model)
}

func toOpenAIMessages(messages []Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		item := map[string]any{"role": string(m.Role)}
		if m.Role == RoleAssistant && m.ReasoningContent != "" {
			item["reasoning_content"] = m.ReasoningContent
		}
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			item["content"] = m.Content
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, c := range m.ToolCalls {
				argsJSON, _ := json.Marshal(c.Arguments)
				calls = append(calls, map[string]any{
					"id":   c.ID,
					"type": "function",
					"function": map[string]any{
						"name":      c.Name,
						"arguments": string(argsJSON),
					},
				})
			}
			item["tool_calls"] = calls
		} else if m.Role == RoleTool {
			item["content"] = m.Content
			item["tool_call_id"] = m.ToolCallID
		} else {
			item["content"] = m.Content
		}
		out = append(out, item)
	}
	return out
}

func toOpenAITools(tools []ToolSchema) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  params,
			},
		})
	}
	return out
}

func parseOpenAIResponse(raw []byte, model string) (*Response, error) {
	var envelope struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}
	msg := envelope.Choices[0].Message
	calls := make([]ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		args, _ := ParseToolArguments(tc.Function.Arguments)
		calls = append(calls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return &Response{
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        calls,
		Usage: TokenUsage{
			PromptTokens:     envelope.Usage.PromptTokens,
			CompletionTokens: envelope.Usage.CompletionTokens,
			Model:            model,
		},
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// BuildProviderFromConfig creates a provider from config fields (no thinking).
func BuildProviderFromConfig(providerName, tokenKey, model string) (Provider, error) {
	return BuildProviderFromLLMFields(providerName, tokenKey, model, nil, "")
}

// BuildProviderFromLLMFields creates a provider with optional thinking settings.
func BuildProviderFromLLMFields(
	providerName, tokenKey, model string,
	thinking *bool,
	reasoningEffort string,
) (Provider, error) {
	if tokenKey == "" {
		return nil, fmt.Errorf("LLM 未配置：请填写 llm.token_key")
	}
	name := ProviderName(providerName)
	if name == "" {
		name = ProviderDeepSeek
	}
	preset, ok := Presets[name]
	if !ok {
		return nil, fmt.Errorf("unknown llm provider: %s", providerName)
	}
	resolved := ResolveModel(name, model)
	effort := strings.TrimSpace(reasoningEffort)
	if effort == "" {
		effort = "high"
	}
	return NewOpenAIProvider(OpenAIOptions{
		Model:           resolved,
		APIKey:          tokenKey,
		BaseURL:         preset.BaseURL,
		ThinkingEnabled: ResolveThinkingEnabled(name, resolved, thinking),
		ReasoningEffort: effort,
	}), nil
}
