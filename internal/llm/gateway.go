package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GatewayConfig controls retry behavior.
type GatewayConfig struct {
	MaxRetries        int
	RetryWait         time.Duration
	Temperature       float64
	MaxTokens         int
}

// Gateway wraps a primary LLM provider with retries and optional fallbacks.
type Gateway struct {
	primary   Provider
	fallbacks []Provider
	config    GatewayConfig
	sleep     func(time.Duration)
}

// NewGateway creates a model gateway.
func NewGateway(primary Provider, cfg GatewayConfig) *Gateway {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryWait == 0 {
		cfg.RetryWait = 5 * time.Second
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.2
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	return &Gateway{
		primary: primary,
		config:  cfg,
		sleep:   time.Sleep,
	}
}

// SetFallbacks wires ordered backup providers (Hermes fallback_providers parity).
func (g *Gateway) SetFallbacks(fallbacks []Provider) {
	if g == nil {
		return
	}
	g.fallbacks = fallbacks
}

// SetSleep replaces sleep for tests.
func (g *Gateway) SetSleep(fn func(time.Duration)) {
	g.sleep = fn
}

// Model returns the primary provider model id.
func (g *Gateway) Model() string {
	if g == nil || g.primary == nil {
		return ""
	}
	return g.primary.Model()
}

// Chat invokes providers with retries; fails over on 401/403/429/5xx.
func (g *Gateway) Chat(ctx context.Context, messages []Message, tools []ToolSchema, sessionID string, step int) (*Response, error) {
	return g.ChatStream(ctx, messages, tools, sessionID, step, StreamHandlerFrom(ctx))
}

// ChatStream is like Chat but forwards token deltas when the provider supports SSE.
// If onDelta is nil and ctx carries a StreamHandler, that handler is used.
func (g *Gateway) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolSchema,
	sessionID string,
	step int,
	onDelta StreamHandler,
) (*Response, error) {
	_ = sessionID
	_ = step
	if ctx == nil {
		ctx = context.Background()
	}
	if onDelta == nil {
		onDelta = StreamHandlerFrom(ctx)
	}
	providers := g.providers()
	var lastErr error
	for i, provider := range providers {
		resp, err := g.chatStreamWithRetries(ctx, provider, messages, tools, onDelta)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if i < len(providers)-1 && FailoverEligible(err) {
			continue
		}
		break
	}
	return nil, fmt.Errorf("LLM gateway failed: %w", lastErr)
}

func (g *Gateway) providers() []Provider {
	if g == nil {
		return nil
	}
	out := make([]Provider, 0, 1+len(g.fallbacks))
	if g.primary != nil {
		out = append(out, g.primary)
	}
	out = append(out, g.fallbacks...)
	return out
}

func (g *Gateway) chatStreamWithRetries(
	ctx context.Context,
	provider Provider,
	messages []Message,
	tools []ToolSchema,
	onDelta StreamHandler,
) (*Response, error) {
	var lastErr error
	for attempt := 0; attempt < g.config.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := invokeProvider(ctx, provider, messages, tools, g.config.Temperature, g.config.MaxTokens, onDelta)
		if err == nil {
			if MalformedToolCallResponse(resp) {
				lastErr = fmt.Errorf("malformed tool_calls response (finish_reason=%q, tools=%d)", resp.FinishReason, len(resp.ToolCalls))
				if attempt < g.config.MaxRetries-1 {
					g.sleep(g.config.RetryWait)
					continue
				}
				return resp, nil
			}
			return resp, nil
		}
		lastErr = err
		if attempt < g.config.MaxRetries-1 && !FailoverEligible(err) {
			g.sleep(g.config.RetryWait)
		}
	}
	return nil, lastErr
}

func invokeProvider(
	ctx context.Context,
	provider Provider,
	messages []Message,
	tools []ToolSchema,
	temperature float64,
	maxTokens int,
	onDelta StreamHandler,
) (*Response, error) {
	if onDelta != nil {
		if s, ok := provider.(Streamer); ok {
			return s.ChatStream(ctx, messages, tools, temperature, maxTokens, onDelta)
		}
	}
	resp, err := provider.Chat(ctx, messages, tools, temperature, maxTokens)
	if err != nil {
		return nil, err
	}
	// Non-streaming provider: deliver the full content once so callers still see a delta.
	if onDelta != nil && resp != nil && resp.Content != "" && len(resp.ToolCalls) == 0 {
		onDelta(StreamDelta{Content: resp.Content})
	}
	return resp, nil
}

// MalformedToolCallResponse detects finish_reason=tool_calls without any tool_calls payload
// (seen with some OpenAI-compatible gateways when the tool list is large).
func MalformedToolCallResponse(resp *Response) bool {
	if resp == nil {
		return false
	}
	if len(resp.ToolCalls) > 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(resp.FinishReason), "tool_calls")
}

// GatewayError is returned when all retries fail.
type GatewayError struct {
	Message string
}

func (e *GatewayError) Error() string { return e.Message }
