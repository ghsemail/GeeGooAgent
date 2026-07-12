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

// Gateway wraps a primary LLM provider with retries.
type Gateway struct {
	primary Provider
	config  GatewayConfig
	sleep   func(time.Duration)
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

// Chat invokes the provider with retries.
func (g *Gateway) Chat(ctx context.Context, messages []Message, tools []ToolSchema, sessionID string, step int) (*Response, error) {
	_ = sessionID
	_ = step
	if ctx == nil {
		ctx = context.Background()
	}
	var lastErr error
	for attempt := 0; attempt < g.config.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := g.primary.Chat(ctx, messages, tools, g.config.Temperature, g.config.MaxTokens)
		if err == nil {
			if MalformedToolCallResponse(resp) {
				lastErr = fmt.Errorf("malformed tool_calls response (finish_reason=%q, tools=%d)", resp.FinishReason, len(resp.ToolCalls))
				if attempt < g.config.MaxRetries-1 {
					g.sleep(g.config.RetryWait)
					continue
				}
				// Final attempt still malformed: return response so caller can surface a clear message.
				return resp, nil
			}
			return resp, nil
		}
		lastErr = err
		if attempt < g.config.MaxRetries-1 {
			g.sleep(g.config.RetryWait)
		}
	}
	return nil, fmt.Errorf("LLM gateway failed: %w", lastErr)
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
