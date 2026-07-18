package agent

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestFilterInteractiveSchemasDropsWorkflowTools(t *testing.T) {
	t.Parallel()
	in := []llm.ToolSchema{
		{Name: "search_code"}, {Name: "read_working_state"}, {Name: "create_pre_market_report"},
	}
	out := filterInteractiveSchemas(in)
	if len(out) != 1 || out[0].Name != "search_code" {
		t.Fatalf("out=%v", out)
	}
}

func TestInterceptToolCallBlocksWorkingStateInChat(t *testing.T) {
	t.Parallel()
	l := &Loop{}
	res, ok := l.interceptToolCall(llm.ToolCall{Name: "read_working_state"}, tools.Context{Interactive: true})
	if !ok || res.Status != tools.StatusSkip {
		t.Fatalf("res=%+v ok=%v", res, ok)
	}
}

func TestInterceptToolCallAllowsWorkflowWhenNotInteractive(t *testing.T) {
	t.Parallel()
	l := &Loop{}
	_, ok := l.interceptToolCall(llm.ToolCall{Name: "read_working_state"}, tools.Context{Interactive: false})
	if ok {
		t.Fatal("expected no intercept for workflow mode")
	}
}

func TestLoopSkipsWorkflowToolInInteractiveChat(t *testing.T) {
	t.Parallel()
	var executed []string
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "read_working_state",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			executed = append(executed, "read_working_state")
			return tools.Result{Status: tools.StatusOK, Summary: "should not run"}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "read_working_state"}}},
			{Content: "已了解，请用 recall。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := NewLoop(gateway, runtime.NewExecutor(registry))
	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "之前查过什么", tools.Context{
		SessionID: session.ID, Interactive: true,
	}, registry.Schemas([]string{"read_working_state"}))

	if len(executed) != 0 {
		t.Fatalf("handler ran: %v", executed)
	}
	if result.Failed {
		t.Fatalf("turn failed: %s", result.Error)
	}
}
