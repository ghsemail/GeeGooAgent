package agent_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// hangingProvider never returns unless the context is cancelled.
type hangingProvider struct{}

func (h *hangingProvider) Model() string { return "hang" }

func (h *hangingProvider) Chat(ctx context.Context, _ []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRunTurnRespectsCancelledContext(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	gateway := llm.NewGateway(&hangingProvider{}, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	session := runtime.NewSession()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	result := loop.RunTurn(ctx, session, "hi", tools.Context{}, nil)
	if !result.Failed {
		t.Fatal("expected failure on cancelled context")
	}
	if !strings.Contains(result.Error, "context canceled") && !strings.Contains(result.Error, "interrupted") {
		t.Fatalf("expected context cancellation error, got: %s", result.Error)
	}
}
