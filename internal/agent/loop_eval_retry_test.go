package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type retryEvaluator struct {
	calls int
}

func (e *retryEvaluator) Evaluate(_ context.Context, in cognition.EvalInput) (cognition.EvalResult, error) {
	e.calls++
	if strings.Contains(in.AssistantText, "改进后") {
		return cognition.EvalResult{Accept: true}, nil
	}
	return cognition.EvalResult{
		Accept: false, RetrySuggested: true, Reason: "回答过于空泛",
	}, nil
}

func TestLoopEvalRetryRerunsAssistantOnce(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "空泛回答"},
			{Content: "改进后的具体回答"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	eval := &retryEvaluator{}
	loop.SetCognition(cognition.Bundle{Evaluator: eval})
	loop.SetEvalMaxRetries(1)

	var events []string
	loop.SetProgress(func(event string, _ map[string]any) {
		events = append(events, event)
	})

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "你好", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("turn failed: %+v", result)
	}
	if result.AssistantText != "改进后的具体回答" {
		t.Fatalf("assistant text = %q", result.AssistantText)
	}
	if eval.calls != 2 {
		t.Fatalf("evaluator calls = %d, want 2", eval.calls)
	}
	if !containsEvent(events, "eval_retry") {
		t.Fatalf("missing eval_retry event: %v", events)
	}
}

func TestLoopEvalRetryDisabledByDefault(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "空泛回答"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetCognition(cognition.Bundle{Evaluator: &retryEvaluator{}})

	session := runtime.NewSession()
	result := loop.RunTurn(context.Background(), session, "你好", tools.Context{}, nil)
	if result.AssistantText != "空泛回答" {
		t.Fatalf("assistant text = %q", result.AssistantText)
	}
}

func containsEvent(events []string, name string) bool {
	for _, e := range events {
		if e == name {
			return true
		}
	}
	return false
}
