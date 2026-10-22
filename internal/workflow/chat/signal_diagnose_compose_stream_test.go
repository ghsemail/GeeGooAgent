package chat

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type streamComposeMock struct {
	llm.MockProvider
	deltas int
}

func (m *streamComposeMock) ChatStream(
	ctx context.Context,
	messages []llm.Message,
	tools []llm.ToolSchema,
	temperature float64,
	maxTokens int,
	onDelta llm.StreamHandler,
) (*llm.Response, error) {
	if onDelta != nil {
		onDelta(llm.StreamDelta{Content: "[[S1]]流式"})
		m.deltas++
		onDelta(llm.StreamDelta{Content: "评价"})
		m.deltas++
	}
	return &llm.Response{Content: "[[S1]]流式评价"}, nil
}

func TestSignalDiagnoseComposeUsesChatStream(t *testing.T) {
	flow := &Flow{
		Template: SkillSignalDiagnose, Phase: PhaseDiagCompose,
		Strategies: []StrategyItem{{Frequency: "60m"}},
		MonthsBack: 3,
	}
	mock := &streamComposeMock{MockProvider: llm.MockProvider{ModelName: "mock"}}
	r := &Runner{ComposeLLM: mock}
	text, err := r.synthesizeSignalDiagnoseJudgment(context.Background(), flow)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "流式评价") || mock.deltas != 2 {
		t.Fatalf("text=%q deltas=%d", text, mock.deltas)
	}
	if flow.ComposeFallback {
		t.Fatal("LLM stream should not set compose_fallback")
	}
	if flow.ComposeTimeoutMs != defaultSignalDiagnoseComposeDeadline.Milliseconds() {
		t.Fatalf("timeout_ms=%d", flow.ComposeTimeoutMs)
	}
}

func TestComposeStreamPublisherThrottlesDuringSlowStream(t *testing.T) {
	flow := &Flow{RunID: "r1", Template: SkillSignalDiagnose}
	state := newComposeStreamState()
	var n int32
	r := &Runner{}
	r.OnProgress = func(event string, _ map[string]any) {
		if event == "eval_judgment_delta" {
			atomic.AddInt32(&n, 1)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	pub := startComposeStreamPublisher(ctx, r, flow, state)
	for i := 0; i < 50; i++ {
		state.ingestDelta(llm.StreamDelta{Content: "x"}, func(string) {})
	}
	pub.finish()
	cancel()
	if atomic.LoadInt32(&n) < 1 {
		t.Fatalf("progress=%d want >=1 final flush", n)
	}
	if state.chunks != 50 {
		t.Fatalf("chunks=%d", state.chunks)
	}
}
