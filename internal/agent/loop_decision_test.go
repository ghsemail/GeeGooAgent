package agent_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/agent/decision"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestLoopDecisionShadowEmitsEvent(t *testing.T) {
	registry := tools.NewRegistry()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "done without tools"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetDecisionCoordinator(&decision.Coordinator{
		Mode: decision.ModeShadow,
		Hook: decision.StubHook{ChoiceID: "reply", Confidence: 0.9},
	})

	var sawDecision bool
	loop.SetProgress(func(event string, data map[string]any) {
		if event == "decision" {
			sawDecision = true
			if data["who"] != string(decision.WhoJev) {
				t.Fatalf("who=%v", data["who"])
			}
			if data["takeover"] == true {
				t.Fatal("shadow must not set takeover")
			}
		}
	})

	result := loop.RunTurn(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if !sawDecision {
		t.Fatal("expected decision SSE event")
	}
}

func TestLoopPlanGateUnchangedWithDecisionShadow(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "create_dca_bot",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "created"}
		},
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{ToolCalls: []llm.ToolCall{{ID: "m1", Name: "create_dca_bot", Arguments: map[string]any{}}}},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetDecisionCoordinator(&decision.Coordinator{
		Mode: decision.ModeShadow,
		Hook: decision.StubHook{ChoiceID: "proceed", Confidence: 1},
	})

	result := loop.RunTurn(context.Background(), runtime.NewSession(), "创建", tools.Context{Interactive: true}, registry.Schemas(nil))
	if !result.PlanPending {
		t.Fatalf("plan gate should still hold; got %+v", result)
	}
}
