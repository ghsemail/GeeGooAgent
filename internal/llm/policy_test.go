package llm_test

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestConfigPolicyPreservesChatDefaults(t *testing.T) {
	t.Parallel()
	p := llm.NewConfigPolicy(llm.ConfigPolicyInput{
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	d := p.Decide(llm.Request{Kind: llm.TaskChat})
	if d.Temperature != 0.3 || d.MaxTokens != 4096 {
		t.Fatalf("chat decision=%+v", d)
	}
	d2 := p.Decide(llm.Request{Kind: llm.TaskSynthesis})
	if d2.Temperature != 0.3 || d2.MaxTokens != 4096 {
		t.Fatalf("synthesis should match chat defaults for behavior parity, got %+v", d2)
	}
}

func TestConfigPolicyCompressUsesDedicatedTemperature(t *testing.T) {
	t.Parallel()
	p := llm.NewConfigPolicy(llm.ConfigPolicyInput{
		Temperature:          0.7,
		MaxTokens:            8192,
		CompressTemperature:  0.2,
		CompressMaxTokens:    1024,
	})
	d := p.Decide(llm.Request{Kind: llm.TaskCompress})
	if d.Temperature != 0.2 || d.MaxTokens != 1024 {
		t.Fatalf("compress=%+v", d)
	}
	if !d.PreferCompress {
		t.Fatal("PreferCompress should be true for TaskCompress")
	}
}

func TestComplexityPolicyRaisesMaxTokens(t *testing.T) {
	t.Parallel()
	base := llm.NewConfigPolicy(llm.ConfigPolicyInput{Temperature: 0.2, MaxTokens: 4096})
	p := llm.ComplexityPolicy{Inner: base, ComplexMinTokens: 8192, ToolSchemaThreshold: 40}
	simple := p.Decide(llm.Request{Kind: llm.TaskChat, ToolSchemaCount: 5})
	if simple.MaxTokens != 4096 {
		t.Fatalf("simple=%+v", simple)
	}
	complex := p.Decide(llm.Request{Kind: llm.TaskComplex, ToolSchemaCount: 5})
	if complex.MaxTokens != 8192 {
		t.Fatalf("complex kind=%+v", complex)
	}
	manyTools := p.Decide(llm.Request{Kind: llm.TaskChat, ToolSchemaCount: 50})
	if manyTools.MaxTokens != 8192 {
		t.Fatalf("many tools=%+v", manyTools)
	}
}

type captureProvider struct {
	temperature float64
	maxTokens   int
}

func (p *captureProvider) Model() string { return "capture" }

func (p *captureProvider) Chat(_ context.Context, _ []llm.Message, _ []llm.ToolSchema, temperature float64, maxTokens int) (*llm.Response, error) {
	p.temperature = temperature
	p.maxTokens = maxTokens
	return &llm.Response{Content: "ok"}, nil
}

func TestGatewayAppliesPolicyFromCallMeta(t *testing.T) {
	t.Parallel()
	provider := &captureProvider{}
	gw := llm.NewGateway(provider, llm.GatewayConfig{
		MaxRetries: 1, RetryWait: time.Millisecond, Temperature: 0.2, MaxTokens: 4096,
	})
	gw.SetPolicy(llm.NewConfigPolicy(llm.ConfigPolicyInput{
		Temperature: 0.2, MaxTokens: 4096,
		CompressTemperature: 0.1, CompressMaxTokens: 512,
	}))
	ctx := llm.WithCallMeta(context.Background(), llm.CallMeta{Kind: llm.TaskCompress})
	_, err := gw.Chat(ctx, nil, nil, "s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if provider.temperature != 0.1 || provider.maxTokens != 512 {
		t.Fatalf("got temp=%v max=%d", provider.temperature, provider.maxTokens)
	}
}
