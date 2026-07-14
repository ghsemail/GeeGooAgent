package report_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/report"
)

func TestSynthesizeParsesJSONAndEnforcesReasonLength(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("引用 [ev_abc] stock.00700.HK.price 腾讯 312.5; ", 6) // >80 chars
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"reason":"` + long + `","suggestion":"hold","summary":"腾讯 312.5，建议持有"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	ws := memory.StockWorkspace{Code: "00700.HK", StockName: "腾讯控股", Attitude: "bullish"}
	ev := []memory.EvidenceRef{{ID: "ev_abc", Source: "stock.00700.HK.price", Summary: "price=312.5"}}
	res, err := synth.Synthesize(context.Background(), ws, ev, memory.MarketContext{})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if !strings.Contains(res.Reason, "ev_abc") {
		t.Fatalf("reason should reference evidence id: %q", res.Reason)
	}
	if res.Suggestion != "hold" {
		t.Fatalf("suggestion = %q", res.Suggestion)
	}
}

func TestSynthesizeRejectsShortReason(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `{"reason":"短","suggestion":"hold","summary":"x"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	_, err := synth.Synthesize(context.Background(), memory.StockWorkspace{}, nil, memory.MarketContext{})
	if err == nil {
		t.Fatal("expected error for short reason")
	}
}

func TestSynthesizeStripsMarkdownFences(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("引用证据 ev_1 详细分析数据支撑结论; ", 6)
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: "```json\n{\"reason\":\"" + long + "\",\"suggestion\":\"buy\",\"summary\":\"ok\"}\n```"},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := report.NewSynthesizer(gateway, "mock")
	res, err := synth.Synthesize(context.Background(), memory.StockWorkspace{}, []memory.EvidenceRef{{ID: "ev_1"}}, memory.MarketContext{})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if res.Suggestion != "buy" {
		t.Fatalf("suggestion = %q", res.Suggestion)
	}
}

func TestSynthesizeUnavailableWhenGatewayNil(t *testing.T) {
	t.Parallel()
	var synth *report.Synthesizer
	if synth.Available() {
		t.Fatal("nil synthesizer should not be available")
	}
}
