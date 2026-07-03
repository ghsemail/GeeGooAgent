package agent_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestAgentRunDelegatesToLoop(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: "hello from agent"},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(gateway, runtime.NewExecutor(registry), registry)
	session := runtime.NewSession()
	result := a.Run(context.Background(), session, "hi", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if result.AssistantText != "hello from agent" {
		t.Fatalf("assistant text = %q", result.AssistantText)
	}
}

func TestAgentSetProgressWiresCallback(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{{Content: "ok"}}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(gateway, runtime.NewExecutor(registry), registry)
	var seen []string
	a.SetProgress(func(event string, _ map[string]any) { seen = append(seen, event) })
	_ = a.Run(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if len(seen) == 0 {
		t.Fatal("progress callback not wired")
	}
}
