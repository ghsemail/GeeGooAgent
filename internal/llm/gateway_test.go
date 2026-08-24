package llm_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestGatewayRetriesMalformedToolCalls(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "", FinishReason: "tool_calls"},
			{Content: "", FinishReason: "tool_calls", ToolCalls: []llm.ToolCall{
				{ID: "c1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}},
			}},
		},
	}
	gw := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 3, RetryWait: time.Millisecond})
	gw.SetSleep(func(time.Duration) {})
	resp, err := gw.Chat(context.Background(), nil, nil, "s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "get_current_price" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestMalformedToolCallResponse(t *testing.T) {
	t.Parallel()
	if !llm.MalformedToolCallResponse(&llm.Response{FinishReason: "tool_calls"}) {
		t.Fatal("expected malformed")
	}
	if llm.MalformedToolCallResponse(&llm.Response{
		FinishReason: "tool_calls",
		ToolCalls:    []llm.ToolCall{{ID: "1", Name: "x"}},
	}) {
		t.Fatal("expected ok")
	}
}

func TestGatewayChatSynthesisFailoverOnParseError(t *testing.T) {
	t.Parallel()
	primary := &llm.MockProvider{
		ModelName: "primary-model",
		Responses: []*llm.Response{{Content: "not json"}},
	}
	fallback := &llm.MockProvider{
		ModelName: "fallback-model",
		Responses: []*llm.Response{{Content: `{"ok":true}`}},
	}
	gw := llm.NewGateway(primary, llm.GatewayConfig{MaxRetries: 1, RetryWait: time.Millisecond})
	gw.SetSleep(func(time.Duration) {})
	gw.SetFallbacks([]llm.Provider{fallback})
	resp, err := gw.ChatSynthesis(context.Background(), nil, func(r *llm.Response) error {
		if strings.TrimSpace(r.Content) != `{"ok":true}` {
			return fmt.Errorf("invalid json")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("got %q", resp.Content)
	}
}

func TestGatewayChatSynthesisFallbackAfterParentDeadline(t *testing.T) {
	t.Parallel()
	primary := &llm.MockProvider{
		ModelName: "primary-model",
		Responses: []*llm.Response{{Content: "<html>bad</html>"}},
	}
	fallback := &llm.MockProvider{
		ModelName: "fallback-model",
		Responses: []*llm.Response{{Content: `{"ok":true}`}},
	}
	gw := llm.NewGateway(primary, llm.GatewayConfig{MaxRetries: 1, RetryWait: time.Millisecond})
	gw.SetFallbacks([]llm.Provider{fallback})
	parentCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)
	resp, err := gw.ChatSynthesis(parentCtx, nil, func(r *llm.Response) error {
		if strings.TrimSpace(r.Content) != `{"ok":true}` {
			return fmt.Errorf("invalid content %q", r.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("got %q", resp.Content)
	}
}

func TestGatewayFailoverOnRateLimit(t *testing.T) {
	t.Parallel()
	primary := &llm.MockProvider{Err: &llm.HTTPError{StatusCode: 429, Body: "rate limit"}}
	fallback := &llm.MockProvider{
		ModelName: "fallback-model",
		Responses: []*llm.Response{{Content: "fallback ok"}},
	}
	gw := llm.NewGateway(primary, llm.GatewayConfig{MaxRetries: 1, RetryWait: time.Millisecond})
	gw.SetSleep(func(time.Duration) {})
	gw.SetFallbacks([]llm.Provider{fallback})
	resp, err := gw.Chat(context.Background(), nil, nil, "s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "fallback ok" {
		t.Fatalf("got %q", resp.Content)
	}
}
