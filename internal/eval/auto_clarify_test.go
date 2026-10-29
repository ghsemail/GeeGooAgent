package eval

import (
	"context"
	"testing"
)

type stubClarifyRecommender struct {
	rec ClarifyRecommendation
}

func (s stubClarifyRecommender) Recommend(
	_ context.Context,
	_ string,
	_ []string,
	_ ClarifyRecommendContext,
) (ClarifyRecommendation, error) {
	return s.rec, nil
}

func TestRecommendClarifyChoiceJEVBeforeScriptDefaults(t *testing.T) {
	rec := RecommendClarifyChoice(
		t.Context(),
		"为腾讯选哪个组合信号？",
		[]string{"SAR+MACD 组合", "RSI 触发型", "EMA 趋势型"},
		ClarifyRecommendContext{
			ClarifyDefaults: []string{"SAR+MACD 组合"},
		},
		stubClarifyRecommender{
			rec: ClarifyRecommendation{
				Index:  2,
				Choice: "EMA 趋势型",
				Reason: "jev pick",
				Source: "jev",
			},
		},
	)
	if rec.Source != "jev" {
		t.Fatalf("source=%q want jev", rec.Source)
	}
	if rec.Index != 2 {
		t.Fatalf("index=%d choice=%q", rec.Index, rec.Choice)
	}
	if rec.AutoPickSeconds != DefaultClarifyAutoPickSeconds {
		t.Fatalf("auto_pick_seconds=%d", rec.AutoPickSeconds)
	}
}

func TestRecommendClarifyChoiceHeuristic(t *testing.T) {
	rec := RecommendClarifyChoice(
		t.Context(),
		"请选择频率",
		[]string{"15m", "60m", "daily"},
		ClarifyRecommendContext{},
		nil,
	)
	answer, ok := rec.AnswerChoice([]string{"15m", "60m", "daily"})
	if !ok || answer != "60m" {
		t.Fatalf("answer=%q ok=%v", answer, ok)
	}
	if rec.Source != "heuristic" {
		t.Fatalf("source=%q", rec.Source)
	}
}

func TestDefaultClarifyAutoPickSeconds(t *testing.T) {
	if DefaultClarifyAutoPickSeconds != 20 {
		t.Fatalf("DefaultClarifyAutoPickSeconds=%d want 20", DefaultClarifyAutoPickSeconds)
	}
}

func TestClarifyTurnWaitPadding(t *testing.T) {
	opts := TurnPlanCaseOptions{
		ExpectMode:   "clarify",
		ClarifyReply: "先问答",
	}.Normalize()
	if got := ClarifyTurnWaitPadding(opts); got != EvalClarifyWaitBudgetSec {
		t.Fatalf("padding=%d want %d", got, EvalClarifyWaitBudgetSec)
	}
}

func TestClarifyRecommendationAnswerChoiceByIndex(t *testing.T) {
	rec := ClarifyRecommendation{Index: 1, Choice: "B"}
	got, ok := rec.AnswerChoice([]string{"A", "B", "C"})
	if !ok || got != "B" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}
