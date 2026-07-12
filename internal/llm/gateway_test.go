package llm_test

import (
	"context"
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
