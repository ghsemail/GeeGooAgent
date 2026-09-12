package agent_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestTurnPlanMultiStockUsesDelegateTasks(t *testing.T) {
	registry := tools.NewRegistry()
	var delegateCalls atomic.Int32
	sub := agent.NewSubAgent(agent.SubAgentConfig{
		Gateway: llm.NewGateway(&llm.MockProvider{
			Responses: []*llm.Response{
				{Content: "腾讯控股现价约 380 港元。"},
				{Content: "阿里巴巴现价约 85 港元。"},
			},
		}, llm.GatewayConfig{MaxRetries: 1}),
		Executor:      runtime.NewExecutor(registry),
		Registry:      registry,
		MaxSteps:      5,
		MaxParallel:   2,
		ChatToolNames: func() []string { return []string{"search_code", "get_current_price"} },
	})
	agent.RegisterDelegateTasks(registry, sub)

	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{
				ID:   "d1",
				Name: "delegate_tasks",
				Arguments: map[string]any{
					"tasks": []any{
						map[string]any{"task": "分析腾讯控股最近股价"},
						map[string]any{"task": "分析阿里巴巴最近股价"},
					},
				},
			}}},
			{Content: "腾讯约 380 港元，阿里巴巴约 85 港元；腾讯近期波动略大于阿里。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	withClassifyPlanner(loop, classifyFixture(map[string]string{
		"请帮我分析下腾讯和阿里巴巴最近的股价": cognition.FormatClassifyJSON("stock_analysis", "gather", "multi_symbol_delegate", "test"),
	}))

	schemas := registry.Schemas([]string{
		"search_code", "get_current_price", "get_mcp_analysis", "delegate_task", "delegate_tasks", "clarify",
	})

	loop.SetProgress(func(event string, data map[string]any) {
		if event == "tool_start" && data["name"] == "delegate_tasks" {
			delegateCalls.Add(1)
		}
	})

	session := runtime.NewSession()
	result := loop.RunTurn(
		context.Background(),
		session,
		"请帮我分析下腾讯和阿里巴巴最近的股价",
		tools.Context{SessionID: session.ID},
		schemas,
	)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if delegateCalls.Load() != 1 {
		t.Fatalf("delegate_tasks calls=%d want 1", delegateCalls.Load())
	}
	if !strings.Contains(result.AssistantText, "腾讯") || !strings.Contains(result.AssistantText, "阿里") {
		t.Fatalf("reply=%q", result.AssistantText)
	}
}
