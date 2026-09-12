package cognition

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestFormatRecentDialogueExcludesCurrentUser(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
		{Role: llm.RoleUser, Content: "帮我看看中际旭创有没有买卖点"},
		{Role: llm.RoleAssistant, Content: "已为你探测 SAR+MACD 买卖点。", ToolCalls: []llm.ToolCall{{Name: "probe_bot_signal_series"}}},
		{Role: llm.RoleUser, Content: "换一个策略"},
	}
	got, has := FormatRecentDialogue(msgs, "换一个策略")
	if !has {
		t.Fatal("expected session history")
	}
	if strings.Contains(got, "换一个策略") {
		t.Fatalf("current user must be excluded: %q", got)
	}
	if !strings.Contains(got, "probe_bot_signal_series") {
		t.Fatalf("expected tool name in dialogue: %q", got)
	}
}

func TestBuildClassifyPromptIncludesSessionContext(t *testing.T) {
	in := PlanInput{
		UserText:       "换一个策略",
		LastDomain:     DomainSignalProbe,
		SessionSummary: "用户在测中际旭创买卖点。",
		RecentDialogue: "user: 帮我看看中际旭创有没有买卖点\nassistant [probe_bot_signal_series]: 已探测。",
		sessionHistory: true,
	}
	prompt := buildClassifyPrompt(in)
	for _, want := range []string{
		"Session summary",
		"Recent session dialogue",
		"probe_bot_signal_series",
		"Last turn domain: signal_probe",
		"User: 换一个策略",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n%s", want, prompt)
		}
	}
}

func TestIntentPlannerSessionContextKeepsProbeOnStrategySwap(t *testing.T) {
	mock := &promptCaptureMock{body: `{"domain":"ambiguous","mode":"clarify","confidence":0.7,"reason":"unsure"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{
		UserText:     "换一个策略",
		LastDomain:   DomainSignalProbe,
		RecentDialogue: "user: 帮我看看中际旭创有没有买卖点\nassistant [probe_bot_signal_series]: 已探测。",
		sessionHistory: true,
	})
	if got.Domain != DomainSignalProbe || got.Mode != ModeExecute {
		t.Fatalf("got %s/%s want signal_probe/execute (%s)", got.Domain, got.Mode, got.Reason)
	}
	if !strings.Contains(mock.lastPrompt, "Recent session dialogue") {
		t.Fatalf("classify prompt must include session dialogue")
	}
}

type promptCaptureMock struct {
	body       string
	lastPrompt string
}

func (m *promptCaptureMock) Model() string { return "prompt-capture" }

func (m *promptCaptureMock) Chat(_ context.Context, msgs []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	if len(msgs) > 0 {
		m.lastPrompt = msgs[0].Content
	}
	return &llm.Response{Content: m.body}, nil
}
