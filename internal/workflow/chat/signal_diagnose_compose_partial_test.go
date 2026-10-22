package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestFinalizeLLMComposeTextKeepsPartialOnDeadline(t *testing.T) {
	flow := &Flow{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	<-ctx.Done()
	text, ok := finalizeLLMComposeText(flow, "已写一半", nil, context.DeadlineExceeded, ctx)
	if !ok || text != "已写一半" {
		t.Fatalf("ok=%v text=%q", ok, text)
	}
	if flow.ComposeFallback {
		t.Fatal("partial stream must not be fallback")
	}
	if flow.PhaseTimingsMs["diag_compose:deadline_hit"] != 1 {
		t.Fatalf("gauge=%v", flow.PhaseTimingsMs)
	}
}

func TestFinalizeLLMComposeTextPreferStreamOverEmptyResponse(t *testing.T) {
	flow := &Flow{}
	text, ok := finalizeLLMComposeText(flow, "流式正文", &llm.Response{Content: ""}, nil, context.Background())
	if !ok || text != "流式正文" || flow.ComposeFallback {
		t.Fatalf("ok=%v text=%q fallback=%v", ok, text, flow.ComposeFallback)
	}
}

func TestComposeFallbackOnlyWhenNoLLM(t *testing.T) {
	flow := &Flow{Template: SkillSignalDiagnose, MonthsBack: 3, ProbeBarCount: 1, Strategies: []StrategyItem{{Frequency: "60m"}}}
	r := &Runner{}
	text, err := r.composeFallbackJudgment(flow, errors.New("boom"))
	if err != nil || text == "" || !flow.ComposeFallback {
		t.Fatalf("err=%v fallback=%v", err, flow.ComposeFallback)
	}
}
