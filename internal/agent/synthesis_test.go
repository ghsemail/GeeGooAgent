package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestReportSynthesizerEmitsEvents(t *testing.T) {
	bus := infra.NewEventBus()
	long := strings.Repeat("引用 [ev_abc] 价格 312.5 详细分析; ", 6)
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"reason":"` + long + `","suggestion":"hold","summary":"持有"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := agent.NewReportSynthesizer(gateway, "mock", bus)
	_, _, _, err := synth.Synthesize(context.Background(), memory.StockWorkspace{Code: "00700.HK"}, nil, memory.MarketContext{})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	var started, completed bool
	for _, rec := range bus.History {
		switch rec.Event {
		case "SynthesisStarted":
			started = true
		case "SynthesisCompleted":
			completed = true
		}
	}
	if !started || !completed {
		t.Fatalf("history=%+v", bus.History)
	}
}

func TestAgentSetGatewayUpdatesReportSynthesizer(t *testing.T) {
	initial := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "x"}}}, llm.GatewayConfig{MaxRetries: 1})
	replacement := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "y"}}}, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(initial, runtime.NewExecutor(registry), registry)
	synth := agent.NewReportSynthesizer(initial, "m1", nil)
	a.SetReportSynthesizer(synth)
	a.SetGateway(replacement)
	if a.Gateway != replacement {
		t.Fatal("gateway not updated")
	}
	if !synth.Available() {
		t.Fatal("synthesizer should remain available")
	}
}
