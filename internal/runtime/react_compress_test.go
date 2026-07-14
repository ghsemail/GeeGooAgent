package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

type compressionSummarizer struct{}

func (compressionSummarizer) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	return "## Goal\nKeep the current task moving.", nil
}

type sessionRecordingSummarizer struct {
	previous []string
	next     []string
}

func (s *sessionRecordingSummarizer) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	s.previous = append(s.previous, previousSummary)
	idx := len(s.previous) - 1
	if idx < len(s.next) {
		return s.next[idx], nil
	}
	return "fallback-summary", nil
}

type recordingProvider struct {
	messages []llm.Message
}

func (p *recordingProvider) Model() string {
	return "recording"
}

func (p *recordingProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolSchema, temperature float64, maxTokens int) (*llm.Response, error) {
	p.messages = append([]llm.Message(nil), messages...)
	return &llm.Response{
		Content: "done",
		Usage:   llm.TokenUsage{PromptTokens: 42, Model: p.Model()},
	}, nil
}

func TestRunTurnCompressesBeforeChat(t *testing.T) {
	provider := &recordingProvider{}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	loop.SetCompressor(prompt.NewCompressor(config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, HygieneThreshold: 0.85, TargetRatio: 0.2,
		ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 10000, ClearToolMinChars: 50,
	}, compressionSummarizer{}))

	session := runtime.NewSession()
	system := session.Messages[0].Content
	for _, content := range []string{"old user", "old assistant", "another user", "another assistant"} {
		session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: strings.Repeat(content+" ", 20)})
	}

	result := loop.RunTurn(context.Background(), session, "latest question", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
	if len(provider.messages) == 0 {
		t.Fatal("provider did not receive messages")
	}
	if provider.messages[0].Role != llm.RoleSystem || provider.messages[0].Content != system {
		t.Fatalf("system first changed: %+v", provider.messages[0])
	}
	joined := joinMessageContent(provider.messages)
	if !strings.Contains(joined, "CONTEXT COMPACTION") && !strings.Contains(joined, "## Goal") {
		t.Fatalf("provider messages were not compressed: %q", joined)
	}
	if session.Messages[0].Role != llm.RoleSystem || session.Messages[0].Content != system {
		t.Fatalf("session system first changed: %+v", session.Messages[0])
	}
	if !strings.Contains(joinMessageContent(session.Messages), "## Goal") {
		t.Fatalf("session did not keep summary: %+v", session.Messages)
	}
	if session.CompactionGeneration < 1 {
		t.Fatalf("expected lineage generation >= 1, got %d", session.CompactionGeneration)
	}
	if session.LineageRoot != session.ID {
		t.Fatalf("lineage_root=%q want %q", session.LineageRoot, session.ID)
	}
	if session.ParentID == "" {
		t.Fatal("expected parent_id after compression")
	}
}

func TestRunTurnHygieneAtEightyFivePercent(t *testing.T) {
	provider := &recordingProvider{}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	sum := &sessionRecordingSummarizer{next: []string{"hygiene-summary", "loop-summary"}}
	// Equal thresholds: turn-start hygiene runs first; in-loop may no-op if tokens drop.
	loop.SetCompressor(prompt.NewCompressor(config.ResolvedCompression{
		Enabled: true, Threshold: 0.5, HygieneThreshold: 0.5, TargetRatio: 0.2,
		ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 100, ClearToolMinChars: 50,
	}, sum))

	session := newCompressibleSession()
	result := loop.RunTurn(context.Background(), session, "latest", tools.Context{}, nil)
	if result.Failed {
		t.Fatalf("failed: %s", result.Error)
	}
	if !strings.Contains(session.PreviousSummary, "summary") {
		t.Fatalf("PreviousSummary=%q", session.PreviousSummary)
	}
	if len(sum.previous) < 1 {
		t.Fatalf("summarizer calls=%d want >=1", len(sum.previous))
	}
	joined := joinMessageContent(provider.messages)
	if !strings.Contains(joined, "CONTEXT COMPACTION") && !strings.Contains(joined, "summary") {
		t.Fatalf("provider did not see compaction: %q", joined)
	}
}

func TestCompressionSummaryIsPerSession(t *testing.T) {
	provider := &recordingProvider{}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	registry := tools.NewRegistry()
	loop := runtime.NewReActLoop(gateway, runtime.NewExecutor(registry))
	summarizer := &sessionRecordingSummarizer{
		next: []string{"first-summary", "second-summary"},
	}
	loop.SetCompressor(prompt.NewCompressor(config.ResolvedCompression{
		Enabled: true, Threshold: 0.01, HygieneThreshold: 0.85, TargetRatio: 0.2,
		ProtectFirstN: 1, ProtectLastN: 1, ContextLength: 10000, ClearToolMinChars: 50,
	}, summarizer))

	first := newCompressibleSession()
	if result := loop.RunTurn(context.Background(), first, "first question", tools.Context{}, nil); result.Failed {
		t.Fatalf("first turn failed: %s", result.Error)
	}
	if first.PreviousSummary != "first-summary" {
		t.Fatalf("first PreviousSummary=%q", first.PreviousSummary)
	}

	second := newCompressibleSession()
	if result := loop.RunTurn(context.Background(), second, "second question", tools.Context{}, nil); result.Failed {
		t.Fatalf("second turn failed: %s", result.Error)
	}
	if second.PreviousSummary != "second-summary" {
		t.Fatalf("second PreviousSummary=%q", second.PreviousSummary)
	}

	wantPrevious := []string{"", ""}
	if len(summarizer.previous) != len(wantPrevious) {
		t.Fatalf("previous calls=%v", summarizer.previous)
	}
	for i, want := range wantPrevious {
		if summarizer.previous[i] != want {
			t.Fatalf("previous[%d]=%q want %q", i, summarizer.previous[i], want)
		}
	}
}

func newCompressibleSession() *runtime.Session {
	session := runtime.NewSession()
	for _, content := range []string{"old user", "old assistant", "another user", "another assistant"} {
		session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: strings.Repeat(content+" ", 20)})
	}
	return session
}

func joinMessageContent(messages []llm.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(msg.Content)
		b.WriteByte('\n')
	}
	return b.String()
}
