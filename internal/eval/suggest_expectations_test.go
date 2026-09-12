package eval_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

type stubPlanner struct {
	byText map[string]cognition.TurnPlan
}

func (s stubPlanner) Plan(in cognition.PlanInput) cognition.TurnPlan {
	if p, ok := s.byText[in.UserText]; ok {
		return p
	}
	return cognition.TurnPlan{Domain: cognition.DomainChat, Mode: cognition.ModeTalk, Act: "qa"}
}

func TestSuggestCaseExpectationsTencentBacktest(t *testing.T) {
	planner := stubPlanner{byText: map[string]cognition.TurnPlan{
		"帮我分析一下腾讯的价格走势": {Domain: cognition.DomainStockAnalysis, Mode: cognition.ModeGather, Act: "analyze"},
		"帮我看看哪些策略适合腾讯":  {Domain: cognition.DomainDCAGrid, Mode: cognition.ModeGather, Act: "dca_grid"},
		"帮我用这些策略回测一下":    {Domain: cognition.DomainBacktestRun, Mode: cognition.ModeExecute, Act: "backtest"},
	}}
	dialogue := []eval.EvalDialogueTurn{
		{Role: "user", Text: "帮我分析一下腾讯的价格走势"},
		{Role: "user", Text: "帮我看看哪些策略适合腾讯"},
		{Role: "user", Text: "帮我用这些策略回测一下", Judge: true},
	}
	got, err := eval.SuggestCaseExpectations(context.Background(), dialogue, planner, nil)
	if err != nil {
		t.Fatalf("SuggestCaseExpectations: %v", err)
	}
	if got.ExpectIntent.Domain != "backtest_run" {
		t.Fatalf("domain=%q want backtest_run", got.ExpectIntent.Domain)
	}
	if len(got.ExpectRouting.RequireTools) == 0 || got.ExpectRouting.RequireTools[0] != "run_strategy_backtest" {
		t.Fatalf("require_tools=%v", got.ExpectRouting.RequireTools)
	}
	if !contains(got.ExpectReply.MustCover, "腾讯") || !contains(got.ExpectReply.MustCover, "回测") {
		t.Fatalf("must_cover=%v", got.ExpectReply.MustCover)
	}
	if len(got.Steps) < 2 {
		t.Fatalf("steps=%v", got.Steps)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
