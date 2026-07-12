package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestAgentRunDelegatesToLoop(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{
		{Content: "hello from agent"},
	}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(gateway, runtime.NewExecutor(registry), registry)
	session := runtime.NewSession()
	result := a.Run(context.Background(), session, "hi", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if result.AssistantText != "hello from agent" {
		t.Fatalf("assistant text = %q", result.AssistantText)
	}
}

func TestAgentSetProgressWiresCallback(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{Responses: []*llm.Response{{Content: "ok"}}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(gateway, runtime.NewExecutor(registry), registry)
	var seen []string
	a.SetProgress(func(event string, _ map[string]any) { seen = append(seen, event) })
	_ = a.Run(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if len(seen) == 0 {
		t.Fatal("progress callback not wired")
	}
}

func TestAgentSetGatewayUpdatesLoop(t *testing.T) {
	t.Parallel()
	initial := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "old"}}}, llm.GatewayConfig{MaxRetries: 1})
	replacement := llm.NewGateway(&llm.MockProvider{Responses: []*llm.Response{{Content: "new"}}}, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(initial, runtime.NewExecutor(registry), registry)

	a.SetGateway(replacement)

	result := a.Run(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if result.AssistantText != "new" {
		t.Fatalf("assistant text = %q", result.AssistantText)
	}
	if a.Gateway != replacement {
		t.Fatal("agent gateway not updated")
	}
}

func TestAgentSetCompressorUpdatesLoop(t *testing.T) {
	t.Parallel()
	provider := &agentRecordingProvider{}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	a := agent.New(gateway, runtime.NewExecutor(registry), registry)
	a.SetCompressor(prompt.NewCompressor(config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, TargetRatio: 0.2,
		ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 100, ClearToolMinChars: 50,
	}, agentSummary{}))

	session := runtime.NewSession()
	for _, content := range []string{"old user", "old assistant", "another user", "another assistant"} {
		session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: strings.Repeat(content+" ", 20)})
	}

	result := a.Run(context.Background(), session, "latest question", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if !strings.Contains(joinAgentMessages(provider.messages), "## Goal") {
		t.Fatalf("compressor was not wired into loop: %+v", provider.messages)
	}
}

type agentSummary struct{}

func (agentSummary) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	return "## Goal\nCompressed from agent.", nil
}

type agentRecordingProvider struct {
	messages []llm.Message
}

func (p *agentRecordingProvider) Model() string {
	return "agent-recording"
}

func (p *agentRecordingProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolSchema, temperature float64, maxTokens int) (*llm.Response, error) {
	p.messages = append([]llm.Message(nil), messages...)
	return &llm.Response{
		Content: "done",
		Usage:   llm.TokenUsage{PromptTokens: 42, Model: p.Model()},
	}, nil
}

func joinAgentMessages(messages []llm.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(msg.Content)
		b.WriteByte('\n')
	}
	return b.String()
}
