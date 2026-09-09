package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

func TestChatSynthesisRetriesUntilSuccess(t *testing.T) {
	t.Parallel()
	restore := SetSynthesisRetryPolicyForTest(0, 2)
	defer restore()

	var sleeps int
	prevSleep := synthesisSleep
	synthesisSleep = func(ctx context.Context, d time.Duration) error {
		sleeps++
		return prevSleep(ctx, d)
	}
	defer func() { synthesisSleep = prevSleep }()

	long := strings.Repeat("引用 [ev_abc] stock.00700.HK.price 腾讯 312.5; ", 6)
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `not-json`},
		{Content: `still-bad`},
		{Content: `{"reason":"` + long + `","suggestion":"hold","summary":"腾讯 312.5，建议持有"}`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := NewSynthesizer(gateway, "mock")

	res, err := synth.Synthesize(context.Background(), memory.StockWorkspace{}, nil, memory.MarketContext{})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if res.Suggestion != "hold" {
		t.Fatalf("suggestion = %q", res.Suggestion)
	}
	if sleeps != 2 {
		t.Fatalf("expected 2 retry sleeps, got %d", sleeps)
	}
}

func TestChatSynthesisFailsAfterMaxRetries(t *testing.T) {
	t.Parallel()
	restore := SetSynthesisRetryPolicyForTest(0, 2)
	defer restore()

	synthesisSleep = func(ctx context.Context, d time.Duration) error { return nil }

	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: `bad`},
		{Content: `bad`},
		{Content: `bad`},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	synth := NewSynthesizer(gateway, "mock")

	_, err := synth.Synthesize(context.Background(), memory.StockWorkspace{}, nil, memory.MarketContext{})
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if !strings.Contains(err.Error(), "llm synthesis failed after 2 retries") {
		t.Fatalf("unexpected error: %v", err)
	}
}
