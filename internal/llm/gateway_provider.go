package llm

import (
	"context"
	"fmt"
)

// ProviderFromGateway adapts a Gateway (primary + fallbacks) to the Provider interface.
// Used for background LLM tasks that should follow ops model-management 主备.
func ProviderFromGateway(gw *Gateway) Provider {
	if gw == nil {
		return nil
	}
	return &gatewayProvider{gw: gw}
}

type gatewayProvider struct {
	gw *Gateway
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
	ctx = WithCallMeta(ctx, CallMeta{Kind: TaskCompress})
	return p.gw.Chat(ctx, messages, tools, "", 0)
}
