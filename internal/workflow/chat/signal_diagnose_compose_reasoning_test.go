package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type reasoningThenAnswerStream struct {
	llm.MockProvider
}

func (m *reasoningThenAnswerStream) ChatStream(
	ctx context.Context,
	_ []llm.Message,
	_ []llm.ToolSchema,
	_ float64,
	_ int,
	onDelta llm.StreamHandler,
) (*llm.Response, error) {
	if onDelta != nil {
		onDelta(llm.StreamDelta{ReasoningContent: "channel reasoning"})
		onDelta(llm.StreamDelta{Content: "<think>inline plan</think>[[S1]]中文评价正文"})
	}
	full := "<think>inline plan</think>[[S1]]中文评价正文"
	return &llm.Response{Content: full, ReasoningContent: "channel reasoning"}, nil
}

func TestComposeStreamExcludesReasoningFromEvalJudgment(t *testing.T) {
	flow := &Flow{
		Template: SkillSignalDiagnose, MonthsBack: 3,
		ProbeBuyHits: 1, ProbeSellHits: 1, ProbeBarCount: 10,
		Strategies: []StrategyItem{{Frequency: "60m"}},
	}
	r := &Runner{ComposeLLM: &reasoningThenAnswerStream{MockProvider: llm.MockProvider{ModelName: "deepseek-v4"}}}
	text, err := r.synthesizeSignalDiagnoseJudgment(context.Background(), flow)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "中文评价正文") || !strings.Contains(flow.EvalJudgment, "### 一、信号触发情况") {
		t.Fatalf("judgment=%q eval=%q", text, flow.EvalJudgment)
	}
	if !strings.Contains(flow.ComposeReasoningDebug, "inline plan") || !strings.Contains(flow.ComposeReasoningDebug, "channel reasoning") {
		t.Fatalf("debug=%q", flow.ComposeReasoningDebug)
	}
	if flow.ComposeFallback {
		t.Fatal("expected LLM path")
	}
	if flow.ComposeModel != "deepseek-v4" {
		t.Fatalf("model=%q", flow.ComposeModel)
	}
}
