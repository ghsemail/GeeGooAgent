package app_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestSynthesisReadyRequiresGatewayAndReportSynthesizer(t *testing.T) {
	t.Parallel()
	a := &app.App{}
	if a.SynthesisReady() {
		t.Fatal("expected not ready on empty app")
	}
	gw := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "x"}}}, llm.GatewayConfig{MaxRetries: 1})
	a.SynthesisGateway = gw
	if a.SynthesisReady() {
		t.Fatal("expected not ready without agent synthesizer")
	}
	registry := tools.NewRegistry()
	ag := agent.New(gw, runtime.NewExecutor(registry), registry)
	ag.SetReportSynthesizer(agent.NewReportSynthesizer(gw, "mock", nil))
	a.Agent = ag
	if !a.SynthesisReady() {
		t.Fatal("expected ready when gateway and report synthesizer are wired")
	}
}

func TestSynthesisReadyFalseWhenGatewayNil(t *testing.T) {
	t.Parallel()
	gw := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "x"}}}, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	ag := agent.New(gw, runtime.NewExecutor(registry), registry)
	ag.SetReportSynthesizer(agent.NewReportSynthesizer(gw, "mock", nil))
	a := &app.App{Agent: ag, SynthesisGateway: nil}
	if a.SynthesisReady() {
		t.Fatal("expected not ready without synthesis gateway")
	}
}
