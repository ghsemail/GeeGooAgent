package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunTurnEmitsTurnEvents(t *testing.T) {
	bus := infra.NewEventBus()
	provider := &llm.MockProvider{Responses: []*llm.Response{{Content: "ok"}}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(tools.NewRegistry()))
	loop.SetEventBus(bus)

	_ = loop.RunTurn(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)

	var started, completed bool
	for _, rec := range bus.History {
		switch rec.Event {
		case "TurnStarted":
			started = true
		case "TurnCompleted":
			completed = true
		}
	}
	if !started || !completed {
		t.Fatalf("bus history=%+v", bus.History)
	}
}

func TestRunTurnToolParallelismCap(t *testing.T) {
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{
					{ID: "c1", Name: "slow", Arguments: map[string]any{}},
					{ID: "c2", Name: "slow", Arguments: map[string]any{}},
					{ID: "c3", Name: "slow", Arguments: map[string]any{}},
				},
			},
			{Content: "done"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "slow",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			time.Sleep(50 * time.Millisecond)
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})

	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	loop.SetToolMaxParallel(1)
	start := time.Now()
	_ = loop.RunTurn(context.Background(), runtime.NewSession(), "go", tools.Context{}, registry.Schemas(nil))
	if time.Since(start) < 140*time.Millisecond {
		t.Fatalf("expected serialized tools with max_parallel=1")
	}
}
