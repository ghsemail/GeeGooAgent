package chat

import (
	"context"
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
		onDelta(llm.StreamDelta{Content: "流式"})
		m.deltas++
		onDelta(llm.StreamDelta{Content: "评价"})
		m.deltas++
	}
	return &llm.Response{Content: "流式评价"}, nil
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
	if text != "流式评价" || mock.deltas != 2 {
		t.Fatalf("text=%q deltas=%d", text, mock.deltas)
	}
}
