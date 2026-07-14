package llm

import (
	"context"
	"strings"
	"testing"
)

func TestParseOpenAIStreamContentAndTools(t *testing.T) {
	t.Parallel()
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"role":"assistant"}}]}`,
		`data: {"choices":[{"delta":{"content":"你好"}}]}`,
		`data: {"choices":[{"delta":{"content":"世界"}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"search_code","arguments":"{\"q\""}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"腾讯\"}"}}]},"finish_reason":"tool_calls"}]}`,
		`data: {"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
		`data: [DONE]`,
		"",
	}, "\n")

	var parts []string
	resp, err := parseOpenAIStream(strings.NewReader(sse), "m", func(d StreamDelta) {
		if d.Content != "" {
			parts = append(parts, d.Content)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(parts, "") != "你好世界" {
		t.Fatalf("deltas=%v", parts)
	}
	if resp.Content != "你好世界" {
		t.Fatalf("content=%q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "search_code" {
		t.Fatalf("tools=%+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].Arguments["q"] != "腾讯" {
		t.Fatalf("args=%v", resp.ToolCalls[0].Arguments)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("finish=%q", resp.FinishReason)
	}
	if resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 5 {
		t.Fatalf("usage=%+v", resp.Usage)
	}
}

func TestGatewayChatStreamUsesStreamer(t *testing.T) {
	t.Parallel()
	provider := &MockProvider{
		Stream: true,
		Responses: []*Response{
			{Content: "ABC"},
		},
	}
	gw := NewGateway(provider, GatewayConfig{MaxRetries: 1})
	var got strings.Builder
	resp, err := gw.ChatStream(context.Background(), nil, nil, "s", 1, func(d StreamDelta) {
		got.WriteString(d.Content)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ABC" || got.String() != "ABC" {
		t.Fatalf("resp=%q deltas=%q", resp.Content, got.String())
	}
}
