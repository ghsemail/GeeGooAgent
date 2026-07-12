package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestReActLoopExecutesToolThenReplies(t *testing.T) {
	provider := &llm.MockProvider{
		ModelName: "gpt-test",
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "c1", Name: "search_code", Arguments: map[string]any{"regex": "腾讯"}},
				},
				Usage: llm.TokenUsage{PromptTokens: 10, CompletionTokens: 5, Model: "gpt-test"},
			},
			{
				Content: "腾讯控股代码 00700.HK。",
				Usage:   llm.TokenUsage{PromptTokens: 8, CompletionTokens: 12, Model: "gpt-test"},
			},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "search_code",
		Description: "search",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{
				Status:  tools.StatusOK,
				Summary: "search_code: 1 item(s)",
				Data:    map[string]any{"items": []map[string]string{{"code": "00700.HK"}}},
			}
		},
	})

	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	session := runtime.NewSession()
	result := loop.RunTurn(
		context.Background(),
		session,
		"查一下腾讯",
		tools.Context{SessionID: session.ID},
		registry.Schemas([]string{"search_code"}),
	)

	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if result.AssistantText != "腾讯控股代码 00700.HK。" {
		t.Fatalf("assistant text = %q", result.AssistantText)
	}
}

func TestReActLoopToolRoundTripWithMCPMock(t *testing.T) {
	// End-to-end within runtime: mock LLM requests search_code, registry uses MCP-shaped result.
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "t1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}},
				},
				Usage: llm.TokenUsage{Model: "mock"},
			},
			{
				Content:   "腾讯现价 99.5 港元。",
				Usage:     llm.TokenUsage{Model: "mock"},
				ToolCalls: nil,
			},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "get_current_price",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{
				Status:  tools.StatusOK,
				Summary: "00700.HK price=99.5",
				Data:    map[string]any{"price": 99.5},
			}
		},
	})

	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "腾讯多少钱", tools.Context{}, registry.Schemas([]string{"get_current_price"}))
	if result.AssistantText != "腾讯现价 99.5 港元。" {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestReActLoopEmptyContentFallsBackToReasoning(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{
			Content: "", ReasoningContent: "结论：腾讯约 380 港元", FinishReason: "stop",
		}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "腾讯价格", tools.Context{}, nil)
	if result.AssistantText != "结论：腾讯约 380 港元" {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestReActLoopEmptyContentLengthHint(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "", FinishReason: "length"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if !strings.Contains(result.AssistantText, "max_tokens") {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestReActLoopStripsSIDOnlyContent(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{
			Content: "[SID=abc123]", FinishReason: "stop",
		}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if !strings.Contains(result.AssistantText, "模型未返回可读内容") {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestReActLoopMalformedToolCallsMessage(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "", FinishReason: "tool_calls"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if !strings.Contains(result.AssistantText, "tool_calls") {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestReActLoopEmptyAfterToolErrorSurfacesTool(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "search_code",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusError, Summary: "HTTP 404 for /searchCode: 404 page not found", ExitCode: 1}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{
				Content: "[SID=only]",
				ToolCalls: []llm.ToolCall{
					{ID: "c1", Name: "search_code", Arguments: map[string]any{"regex": "腾讯"}},
				},
			},
			{Content: "", FinishReason: "stop"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	result := loop.RunTurn(
		context.Background(), runtime.NewSession(), "腾讯价格", tools.Context{},
		registry.Schemas([]string{"search_code"}),
	)
	if !strings.Contains(result.AssistantText, "search_code") || !strings.Contains(result.AssistantText, "404") {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestReActLoopSlimRetryAfterMalformedToolCalls(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "search_code",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "search_code: 1 item(s); top: 腾讯控股 (00700.HK)",
				Data: map[string]any{"items": []any{map[string]any{"code": "00700.HK", "name": "腾讯控股"}}}}
		},
	})
	registry.Register(tools.Tool{
		Name: "get_current_price",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "00700.HK = 380",
				Data: map[string]any{"code": "00700.HK", "price": 380}}
		},
	})
	// Extra unused schemas to trigger slim path (len(slim) < len(schemas)).
	extra := []llm.ToolSchema{{
		Name: "create_smart_trade", Description: "create",
		Parameters: map[string]any{"type": "object", "properties": map[string]any{}},
	}}
	schemas := append(registry.Schemas([]string{"search_code", "get_current_price"}), extra...)
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "search_code", Arguments: map[string]any{"regex": "腾讯"}}}},
			{Content: "", FinishReason: "tool_calls"}, // malformed → slim retry
			{ToolCalls: []llm.ToolCall{{ID: "c2", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}}}},
			{Content: "腾讯现价约 380 港元。", FinishReason: "stop"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "腾讯价格", tools.Context{}, schemas)
	if !strings.Contains(result.AssistantText, "380") {
		t.Fatalf("got %q", result.AssistantText)
	}
}
