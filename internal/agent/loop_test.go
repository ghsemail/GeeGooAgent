package agent_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
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

	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
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

	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
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
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
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
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
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
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
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
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
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
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
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
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "腾讯价格", tools.Context{}, schemas)
	if !strings.Contains(result.AssistantText, "380") {
		t.Fatalf("got %q", result.AssistantText)
	}
}

func TestRunTurnInjectsBudgetWarningNearCap(t *testing.T) {
	var seen [][]llm.Message
	provider := &recordingChatProvider{
		onChat: func(messages []llm.Message) *llm.Response {
			seen = append(seen, append([]llm.Message(nil), messages...))
			n := len(seen)
			if n < 3 {
				return &llm.Response{
					ToolCalls: []llm.ToolCall{{ID: fmt.Sprintf("c%d", n), Name: "noop", Arguments: map[string]any{}}},
				}
			}
			return &llm.Response{Content: "最终答复"}
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "noop", Description: "noop",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetMaxToolRounds(3)
	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "go", tools.Context{}, registry.Schemas(nil))
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if len(seen) < 2 {
		t.Fatalf("expected multiple LLM calls, got %d", len(seen))
	}
	found := false
	for _, m := range seen[1] {
		if m.Role == llm.RoleUser && strings.Contains(m.Content, "[BUDGET]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected budget warning on near-cap round, messages=%+v", seen[1])
	}
	for _, m := range session.Messages {
		if strings.Contains(m.Content, "[BUDGET]") {
			t.Fatalf("budget warning must not persist in session: %+v", m)
		}
	}
}

type recordingChatProvider struct {
	onChat func([]llm.Message) *llm.Response
}

func (p *recordingChatProvider) Model() string { return "rec" }

func (p *recordingChatProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolSchema, temperature float64, maxTokens int) (*llm.Response, error) {
	if p.onChat == nil {
		return &llm.Response{Content: "ok"}, nil
	}
	return p.onChat(messages), nil
}

func TestRunTurnRespectsMaxToolRoundsConfig(t *testing.T) {
	provider := &llm.MockProvider{
		ModelName: "gpt-test",
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "noop", Arguments: map[string]any{}}}},
		},
	}
	// Always return another tool call so the loop never finishes naturally.
	provider.Responses = make([]*llm.Response, 5)
	for i := range provider.Responses {
		provider.Responses[i] = &llm.Response{
			ToolCalls: []llm.ToolCall{{ID: fmt.Sprintf("c%d", i), Name: "noop", Arguments: map[string]any{}}},
		}
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "noop", Description: "noop",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetMaxToolRounds(3)
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "go", tools.Context{}, registry.Schemas(nil))
	if !result.Failed || result.Error != "max_tool_rounds" {
		t.Fatalf("expected max_tool_rounds failure, got failed=%v err=%q", result.Failed, result.Error)
	}
}

func TestRunTurnParallelToolCallsPreserveOrder(t *testing.T) {
	var mu sync.Mutex
	active := 0
	maxActive := 0
	provider := &llm.MockProvider{
		ModelName: "gpt-test",
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "c1", Name: "slow_a", Arguments: map[string]any{}},
					{ID: "c2", Name: "slow_b", Arguments: map[string]any{}},
				},
			},
			{Content: "done"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	slow := func(label string) tools.Handler {
		return func(ctx tools.Context, args map[string]any) tools.Result {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()
			time.Sleep(40 * time.Millisecond)
			mu.Lock()
			active--
			mu.Unlock()
			return tools.Result{Status: tools.StatusOK, Summary: label}
		}
	}
	registry.Register(tools.Tool{Name: "slow_a", Description: "a", Handle: slow("A")})
	registry.Register(tools.Tool{Name: "slow_b", Description: "b", Handle: slow("B")})

	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "parallel", tools.Context{}, registry.Schemas(nil))
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if maxActive < 2 {
		t.Fatalf("expected concurrent tool execution, maxActive=%d", maxActive)
	}
	// Tool messages must follow the model's tool_calls order.
	var toolBodies []string
	for _, m := range session.Messages {
		if m.Role != llm.RoleTool {
			continue
		}
		toolBodies = append(toolBodies, m.Content)
	}
	if len(toolBodies) != 2 {
		t.Fatalf("tool messages = %d, want 2", len(toolBodies))
	}
	if !strings.Contains(toolBodies[0], `"summary":"A"`) || !strings.Contains(toolBodies[1], `"summary":"B"`) {
		t.Fatalf("tool order not preserved: %v", toolBodies)
	}
}

func TestRunTurnApprovalCallback(t *testing.T) {
	called := false
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "create_dca_bot",
		Description: "create bot",
		Handle: tools.ApprovalGate("create_dca_bot", func(ctx tools.Context, args map[string]any) tools.Result {
			called = true
			return tools.Result{Status: tools.StatusOK, Summary: "created"}
		}),
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "create_dca_bot", Arguments: map[string]any{"name": "test"}}}},
			{Content: "已创建"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetApproval(func(toolName string, args map[string]any) bool { return true })
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "创建", tools.Context{Interactive: true}, registry.Schemas(nil))
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if !called {
		t.Fatal("expected mutating handler to run after approval")
	}
}

func TestRunTurnApprovalDenied(t *testing.T) {
	called := false
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "delete_smart_trade",
		Description: "delete",
		Handle: tools.ApprovalGate("delete_smart_trade", func(ctx tools.Context, args map[string]any) tools.Result {
			called = true
			return tools.Result{Status: tools.StatusOK, Summary: "deleted"}
		}),
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "delete_smart_trade", Arguments: map[string]any{}}}},
			{Content: "已跳过"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetApproval(func(toolName string, args map[string]any) bool { return false })
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "删除", tools.Context{Interactive: true}, registry.Schemas(nil))
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if called {
		t.Fatal("handler should not run when approval denied")
	}
}
