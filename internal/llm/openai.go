package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolSchema, temperature float64, maxTokens int) (*Response, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
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
			content := scrubSIDTokens(m.Content)
			if content == "" {
				item["content"] = nil
			} else {
				item["content"] = content
			}
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
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content          any    `json:"content"`
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
	choice := envelope.Choices[0]
	msg := choice.Message
	calls := make([]ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		args, err := ParseToolArguments(tc.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("invalid arguments for tool %q: %w", tc.Function.Name, err)
		}
		calls = append(calls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return &Response{
		Content:          coerceMessageContent(msg.Content),
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        calls,
		FinishReason:     choice.FinishReason,
		Usage: TokenUsage{
			PromptTokens:     envelope.Usage.PromptTokens,
			CompletionTokens: envelope.Usage.CompletionTokens,
			Model:            model,
		},
	}, nil
}

// coerceMessageContent accepts string content or OpenAI-style multipart arrays.
func coerceMessageContent(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, part := range v {
			m, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := m["text"].(string); t != "" {
				b.WriteString(t)
			}
		}
		return b.String()
	default:
		return fmt.Sprint(v)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

var sidTokenRE = regexp.MustCompile(`(?i)\[SID=[^\]]+\]`)

func scrubSIDTokens(s string) string {
	return strings.TrimSpace(sidTokenRE.ReplaceAllString(s, ""))
}

// BuildProviderFromConfig creates a provider from config fields (no thinking).
func BuildProviderFromConfig(providerName, tokenKey, model string) (Provider, error) {
	return BuildProviderFromLLMFields(providerName, tokenKey, model, nil, "", "")
}

// BuildProviderFromLLMFields creates a provider with optional thinking settings.
// baseURLOverride, when non-empty, replaces the provider preset BaseURL (ops configured).
func BuildProviderFromLLMFields(
	providerName, tokenKey, model string,
	thinking *bool,
	reasoningEffort string,
	baseURLOverride string,
) (Provider, error) {
	if tokenKey == "" {
		return nil, fmt.Errorf("LLM 未配置：请填写 llm.token_key 或运营配置 token")
	}
	name := ProviderName(providerName)
	if name == "" {
		name = ProviderDeepSeek
	}
	preset, ok := Presets[name]
	if !ok {
		// Unknown provider name (e.g. skimtoken display) → OpenAI-compatible HTTP
		name = ProviderOpenAI
		preset = Presets[ProviderOpenAI]
	}
	resolved := ResolveModel(name, model)
	if strings.TrimSpace(model) != "" {
		// Prefer exact ops model id even if not in preset list
		resolved = strings.TrimSpace(model)
	}
	effort := strings.TrimSpace(reasoningEffort)
	if effort == "" {
		effort = "high"
	}
	baseURL := strings.TrimRight(strings.TrimSpace(baseURLOverride), "/")
	if baseURL == "" {
		baseURL = preset.BaseURL
	}
	return NewOpenAIProvider(OpenAIOptions{
		Model:           resolved,
		APIKey:          tokenKey,
		BaseURL:         baseURL,
		ThinkingEnabled: ResolveThinkingEnabled(name, resolved, thinking),
		ReasoningEffort: effort,
	}), nil
}

// InferProviderFromNames maps ops display_name/name to a preset provider key.
func InferProviderFromNames(displayName, name string) string {
	text := strings.ToLower(strings.TrimSpace(displayName + " " + name))
	switch {
	case strings.Contains(text, "deepseek"):
		return string(ProviderDeepSeek)
	case strings.Contains(text, "minimax"):
		return string(ProviderMinimax)
	default:
		return string(ProviderOpenAI)
	}
}
