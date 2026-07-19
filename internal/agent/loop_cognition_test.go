package agent_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type recordingEvaluator struct {
	calls int
}

func (e *recordingEvaluator) Evaluate(_ context.Context, in cognition.EvalInput) (cognition.EvalResult, error) {
	e.calls++
	if in.AssistantText == "" {
		return cognition.EvalResult{Accept: false, Reason: "empty"}, nil
	}
	return cognition.EvalResult{Accept: true}, nil
}

type neverHoldPolicy struct {
	cognition.DefaultPlanPolicy
}

func (neverHoldPolicy) ShouldHold(cognition.PlanHoldInput) bool { return false }

func TestLoopUsesInjectedPlanPolicyAndEvaluator(t *testing.T) {
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
				Content:   "准备创建",
				ToolCalls: []llm.ToolCall{{ID: "c1", Name: "create_dca_bot", Arguments: map[string]any{"botname": "t"}}},
			},
			{Content: "已完成"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetPlanGate(true)
	eval := &recordingEvaluator{}
	loop.SetCognition(cognition.Bundle{
		Ranker:     cognition.IdentityRanker{},
		Evaluator:  eval,
		PlanPolicy: neverHoldPolicy{},
	})

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "创建 bot", tools.Context{Interactive: true}, registry.Schemas(nil))
	if result.PlanPending {
		t.Fatal("custom PlanPolicy should not hold")
	}
	if !called {
		t.Fatal("mutating tool should run immediately under neverHoldPolicy")
	}
	if result.Failed {
		t.Fatalf("turn failed: %+v", result)
	}
	if eval.calls < 1 {
		t.Fatalf("evaluator not called, calls=%d", eval.calls)
	}
}

func TestLoopRankItemsUsesInjectedRanker(t *testing.T) {
	t.Parallel()
	loop := agent.NewLoop(nil, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetCognition(cognition.Bundle{
		Ranker: reverseRanker{},
	})
	out, err := loop.RankItems(context.Background(), []cognition.RankItem{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0].ID != "c" || out[2].ID != "a" {
		t.Fatalf("got %+v", out)
	}
}

type reverseRanker struct{}

func (reverseRanker) Rank(_ context.Context, items []cognition.RankItem) ([]cognition.RankItem, error) {
	out := make([]cognition.RankItem, len(items))
	for i := range items {
		out[len(items)-1-i] = items[i]
	}
	return out, nil
}
