package agent_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunTurnBudgetExhaustedRequestsSummary(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "noop", Arguments: map[string]any{}}}},
			{ToolCalls: []llm.ToolCall{{ID: "c2", Name: "noop", Arguments: map[string]any{}}}},
			{Content: "阶段性结论：已查询 noop，建议缩小范围。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "noop",
		Handle: func(tools.Context, map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetMaxToolRounds(2)
	result := loop.RunTurn(context.Background(), runtime.NewSession(), "go", tools.Context{}, registry.Schemas(nil))
	if !result.Failed || result.Error != "max_tool_rounds" {
		t.Fatalf("failed=%v err=%q", result.Failed, result.Error)
	}
	if !strings.Contains(result.AssistantText, "阶段性结论") {
		t.Fatalf("assistant=%q", result.AssistantText)
	}
}
