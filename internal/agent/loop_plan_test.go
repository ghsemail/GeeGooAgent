package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestPlanGateHoldsMutatingUntilApproval(t *testing.T) {
	t.Parallel()
	called := false
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "create_dca_bot",
		Handle: tools.ApprovalGate("create_dca_bot", func(ctx tools.Context, args map[string]any) tools.Result {
			called = true
			return tools.Result{Status: tools.StatusOK, Summary: "created"}
		}),
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{
				Content:   "准备创建 DCA bot",
				ToolCalls: []llm.ToolCall{{ID: "c1", Name: "create_dca_bot", Arguments: map[string]any{"botname": "t"}}},
			},
			{Content: "已完成"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetPlanGate(true)
	session := runtime.NewSession()

	result := loop.RunTurn(context.Background(), session, "创建 bot", tools.Context{Interactive: true}, registry.Schemas(nil))
	if !result.PlanPending {
		t.Fatalf("expected plan pending, got %+v", result)
	}
	if called {
		t.Fatal("mutating handler ran before approval")
	}
	if session.PendingPlan == nil || len(session.PendingPlan.ToolCalls) != 1 {
		t.Fatalf("pending plan=%+v", session.PendingPlan)
	}

	result = loop.RunTurn(context.Background(), session, "y", tools.Context{Interactive: true}, registry.Schemas(nil))
	if result.PlanPending || result.Failed {
		t.Fatalf("resume failed: %+v", result)
	}
	if !called {
		t.Fatal("mutating handler should run after y")
	}
	if !strings.Contains(result.AssistantText, "已完成") {
		t.Fatalf("assistant=%q", result.AssistantText)
	}
}

func TestPlanGateRejectsPendingPlan(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "delete_smart_trade",
		Handle: tools.ApprovalGate("delete_smart_trade", func(ctx tools.Context, args map[string]any) tools.Result {
			t.Fatal("should not run")
			return tools.Result{Status: tools.StatusOK}
		}),
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "delete_smart_trade"}}},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetPlanGate(true)
	session := runtime.NewSession()
	_ = loop.RunTurn(context.Background(), session, "删除", tools.Context{Interactive: true}, registry.Schemas(nil))

	result := loop.RunTurn(context.Background(), session, "n", tools.Context{Interactive: true}, registry.Schemas(nil))
	if !strings.Contains(result.AssistantText, "取消") {
		t.Fatalf("result=%q", result.AssistantText)
	}
	if session.PendingPlan != nil {
		t.Fatal("pending plan should be cleared")
	}
}
