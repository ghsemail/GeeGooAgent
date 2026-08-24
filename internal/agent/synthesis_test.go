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
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

var _ workflow.StockPreMarketSynthesizerProvider = (*agent.ReportSynthesizer)(nil)

func TestReportSynthesizerEmitsEvents(t *testing.T) {
	bus := infra.NewEventBus()
	long := strings.Repeat("引用 [ev_abc] 价格 312.5 详细分析; ", 6)
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"reason":"` + long + `","suggestion":"hold","summary":"持有"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := agent.NewReportSynthesizer(gateway, "mock", bus)
	_, err := synth.Synthesize(context.Background(), memory.StockWorkspace{Code: "00700.HK"}, nil, memory.MarketContext{})
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

func TestAgentSetGatewayDoesNotUpdateReportSynthesizer(t *testing.T) {
	initial := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "x"}}}, llm.GatewayConfig{MaxRetries: 1})
	synthGW := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: `{"reason":"` + strings.Repeat("a", 80) + `","suggestion":"hold","summary":"持有"}`}}}, llm.GatewayConfig{MaxRetries: 1})
	replacement := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "y"}}}, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(initial, runtime.NewExecutor(registry), registry)
	synth := agent.NewReportSynthesizer(synthGW, "m1", nil)
	a.SetReportSynthesizer(synth)
	a.SetGateway(replacement)
	if a.Gateway != replacement {
		t.Fatal("gateway not updated")
	}
	if !synth.Available() {
		t.Fatal("synthesizer should remain available")
	}
}

func TestReportSynthesizerSynthesizeStockPreMarket(t *testing.T) {
	reason := strings.Repeat("依据周线与资金流判断 ", 12)
	provider := &llm.MockProvider{Responses: []*llm.Response{{
		Content: `{"report":"## 市场背景\n\n正文","result":"long","confidence":"high","reason":"` + reason + `","suggestion":"hold","summary":"持有观望"}`,
	}}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := agent.NewReportSynthesizer(gateway, "mock", nil)
	ctx := workflow.ContextWithSynthesizer(context.Background(), synth)
	got := workflow.StockPreMarketSynthesizerFrom(ctx)
	if got == nil {
		t.Fatal("ReportSynthesizer should implement StockPreMarketSynthesizerProvider")
	}
	res, err := got.SynthesizeStockPreMarket(ctx, memory.StockWorkspace{Code: "00700.HK"}, "draft", nil, memory.MarketContext{}, "市场摘要", "")
	if err != nil {
		t.Fatalf("synthesize stock premarket: %v", err)
	}
	if res.Result != "long" || res.Summary != "持有观望" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
