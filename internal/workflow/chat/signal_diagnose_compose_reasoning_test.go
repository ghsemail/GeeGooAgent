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
		onDelta(llm.StreamDelta{ReasoningContent: "english planning must not show"})
		onDelta(llm.StreamDelta{Content: "中文评价正文"})
	}
	return &llm.Response{Content: "中文评价正文", ReasoningContent: "english planning must not show"}, nil
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
	if text != "中文评价正文" || flow.EvalJudgment != "中文评价正文" {
		t.Fatalf("judgment=%q eval=%q", text, flow.EvalJudgment)
	}
	if !strings.Contains(flow.ComposeReasoningDebug, "english planning") {
		t.Fatalf("debug=%q", flow.ComposeReasoningDebug)
	}
	if flow.ComposeFallback {
		t.Fatal("expected LLM path")
	}
	if flow.ComposeModel != "deepseek-v4" {
		t.Fatalf("model=%q", flow.ComposeModel)
	}
}
