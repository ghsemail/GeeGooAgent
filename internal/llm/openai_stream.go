package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatStream calls the OpenAI-compatible endpoint with stream=true and assembles
// a full Response while forwarding content deltas to onDelta.
func (p *OpenAIProvider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolSchema,
	temperature float64,
	maxTokens int,
	onDelta StreamHandler,
) (*Response, error) {
	body := map[string]any{
		"model":       p.model,
		"messages":    toOpenAIMessages(messages),
		"temperature": temperature,
		"max_tokens":  maxTokens,
		"stream":      true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
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
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	return parseOpenAIStream(resp.Body, p.model, onDelta)
}

type streamToolAcc struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

func (acc *streamToolAcc) appendArgs(frag string) {
	if acc == nil || frag == "" {
		return
	}
	cur := acc.Arguments.String()
	// Some providers (notably DeepSeek thinking+stream) occasionally resend a
	// complete JSON object instead of a delta fragment. Appending would yield
	// `{"a":1}{"a":1}` and break json.Unmarshal.
	if cur != "" && looksLikeJSONObjectStart(frag) && isCompleteJSONObject(cur) {
		acc.Arguments.Reset()
	}
	acc.Arguments.WriteString(frag)
}

func looksLikeJSONObjectStart(s string) bool {
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

func isCompleteJSONObject(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s[0] != '{' {
		return false
	}
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}

func parseOpenAIStream(r io.Reader, model string, onDelta StreamHandler) (*Response, error) {
	scanner := bufio.NewScanner(r)
	// Tool argument fragments can be large; raise the token limit.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var (
		content   strings.Builder
		reasoning strings.Builder
		finish    string
		usage     TokenUsage
		tools     = map[int]*streamToolAcc{}
	)
	usage.Model = model

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var envelope struct {
			Choices []struct {
				FinishReason string `json:"finish_reason"`
				Delta        struct {
					Content          any    `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
			continue
		}
		if envelope.Usage != nil {
			usage.PromptTokens = envelope.Usage.PromptTokens
			usage.CompletionTokens = envelope.Usage.CompletionTokens
		}
		if len(envelope.Choices) == 0 {
			continue
		}
		choice := envelope.Choices[0]
		if choice.FinishReason != "" {
			finish = choice.FinishReason
		}
		delta := choice.Delta
		text := scrubSIDTokens(coerceMessageContent(delta.Content))
		if text != "" {
			content.WriteString(text)
			if onDelta != nil {
				onDelta(StreamDelta{Content: text})
			}
		}
		if delta.ReasoningContent != "" {
			reasoning.WriteString(delta.ReasoningContent)
			if onDelta != nil {
				onDelta(StreamDelta{ReasoningContent: delta.ReasoningContent})
			}
		}
		for _, tc := range delta.ToolCalls {
			acc, ok := tools[tc.Index]
			if !ok {
				acc = &streamToolAcc{}
				tools[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Function.Name != "" {
				acc.Name = tc.Function.Name
			}
			acc.appendArgs(tc.Function.Arguments)
			if onDelta != nil && (tc.Function.Name != "" || tc.Function.Arguments != "") {
				onDelta(StreamDelta{ToolCall: &ToolCallStreamDelta{
					Index:     tc.Index,
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	calls := make([]ToolCall, 0, len(tools))
	maxIdx := -1
	for i := range tools {
		if i > maxIdx {
			maxIdx = i
		}
	}
	for i := 0; i <= maxIdx; i++ {
		acc := tools[i]
		if acc == nil {
			continue
		}
		args, err := ParseToolArguments(acc.Arguments.String())
		if err != nil {
			return nil, fmt.Errorf("invalid arguments for tool %q: %w", acc.Name, err)
		}
		calls = append(calls, ToolCall{ID: acc.ID, Name: acc.Name, Arguments: args})
	}

	return &Response{
		Content:          content.String(),
		ReasoningContent: reasoning.String(),
		ToolCalls:        calls,
		FinishReason:     finish,
		Usage:            usage,
	}, nil
}
