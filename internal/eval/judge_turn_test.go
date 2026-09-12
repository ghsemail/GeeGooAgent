package eval

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestVerifyTurnPlanIntentUsesJudgedTurn(t *testing.T) {
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_plan_trace": []chatsession.TurnPlanTraceEntry{
				{Turn: 1, Plan: chatsession.TurnPlanSnapshot{Domain: "stock_analysis", Mode: "gather"}},
				{Turn: 2, Plan: chatsession.TurnPlanSnapshot{Domain: "dca_grid", Mode: "gather"}},
				{Turn: 3, Plan: chatsession.TurnPlanSnapshot{Domain: "backtest_run", Mode: "execute"}},
			},
		},
	}
	opts := TurnPlanCaseOptions{
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我分析一下腾讯的价格走势"},
			{Role: "user", Text: "帮我看看哪些策略适合腾讯"},
			{Role: "user", Text: "帮我用这些策略回测一下", Judge: true},
		},
		ExpectIntent: &ExpectIntentSpec{Domain: "backtest_run", Mode: "execute"},
	}.Normalize()

	res := verifyTurnPlanIntent(chat, opts.intent(), opts)
	if !res.Passed {
		t.Fatalf("intent should use judged turn plan: %s", res.Detail)
	}
}
