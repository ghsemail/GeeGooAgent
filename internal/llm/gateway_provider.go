package llm

import (
	"context"
	"fmt"
)

// ProviderFromGateway adapts a Gateway (primary + fallbacks) to the Provider interface.
// Used for background LLM tasks that should follow ops model-management 主备.
// Calls use TaskCompress policy (legacy behavior).
func ProviderFromGateway(gw *Gateway) Provider {
	if gw == nil {
		return nil
	}
	return &gatewayProvider{gw: gw, kind: TaskCompress}
}

// ClassifyProviderFromGateway adapts a Gateway for per-turn intent classification.
// Uses TaskClassify policy (low temperature, dedicated token budget) — same gateway
// as chat so Dock/user model swaps stay in sync.
func ClassifyProviderFromGateway(gw *Gateway) Provider {
	if gw == nil {
		return nil
	}
	return &gatewayProvider{gw: gw, kind: TaskClassify}
}

type gatewayProvider struct {
	gw   *Gateway
	kind TaskKind
}

func (p *gatewayProvider) Model() string {
	if p == nil || p.gw == nil {
		return ""
	}
	return p.gw.Model()
}

func (p *gatewayProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolSchema,
	_ float64,
	_ int,
) (*Response, error) {
	if p == nil || p.gw == nil {
		return nil, fmt.Errorf("gateway provider not configured")
	}
	kind := p.kind
	if kind == "" {
		kind = TaskCompress
	}
	ctx = WithCallMeta(ctx, CallMeta{Kind: kind})
	return p.gw.Chat(ctx, messages, tools, "", 0)
}
