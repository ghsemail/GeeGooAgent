package eval_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestVerifyExecutionProfilesMatchLiveFailures(t *testing.T) {
	technicalChat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_current_price"}},
				{Turn: 2, Tools: []string{"get_single_prompt_template", "get_mcp_analysis"}},
			},
		},
	}
	res := eval.VerifyExecution(technicalChat, eval.ExpectExecutionSpec{
		Profile: domaincatalog.ProfileStockTechnicalFull,
	}, eval.TurnPlanCaseOptions{})
	if !res.Passed {
		t.Fatalf("technical chain should pass profile check: %s", res.Detail)
	}

	colloquialChat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_mcp_analysis"}},
				{Turn: 2, Tools: []string{"get_single_prompt_template", "get_current_price", "get_mcp_analysis"}},
			},
		},
	}
	res = eval.VerifyExecution(colloquialChat, eval.ExpectExecutionSpec{
		Profile: domaincatalog.ProfileStockContextFollowup,
	}, eval.TurnPlanCaseOptions{})
	if !res.Passed {
		t.Fatalf("colloquial ref should pass profile check: %s", res.Detail)
	}

	priceChat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_current_price"}},
			},
		},
	}
	res = eval.VerifyExecution(priceChat, eval.ExpectExecutionSpec{
		Profile: domaincatalog.ProfileStockPriceSnapshot,
	}, eval.TurnPlanCaseOptions{})
	if !res.Passed {
		t.Fatalf("price snapshot should pass profile check: %s", res.Detail)
	}
}

func TestVerifyExecutionUsesJudgedTurnIndex(t *testing.T) {
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_mcp_analysis"}},
				{Turn: 2, Tools: []string{"get_signal_combinations", "clarify"}},
				{Turn: 3, Tools: []string{"run_strategy_backtest"}},
			},
			"turn_plan_trace": []chatsession.TurnPlanTraceEntry{
				{Turn: 1, Plan: chatsession.TurnPlanSnapshot{Domain: "stock_analysis", Mode: "gather"}},
				{Turn: 2, Plan: chatsession.TurnPlanSnapshot{Domain: "dca_grid", Mode: "gather"}},
				{Turn: 3, Plan: chatsession.TurnPlanSnapshot{Domain: "backtest_run", Mode: "execute"}},
			},
		},
	}
	opts := eval.TurnPlanCaseOptions{
		Dialogue: []eval.EvalDialogueTurn{
			{Role: "user", Text: "帮我分析一下腾讯的价格走势"},
			{Role: "user", Text: "帮我看看哪些策略适合腾讯"},
			{Role: "user", Text: "组合信号（多指标共振，推荐稳健）", OnClarify: true},
			{Role: "user", Text: "帮我用这些策略回测一下", Judge: true},
		},
		ExpectIntent: &eval.ExpectIntentSpec{Domain: "backtest_run", Mode: "execute"},
	}.Normalize()

	res := eval.VerifyExecution(chat, eval.ExpectExecutionSpec{
		LegacyRequireTools: []string{"run_strategy_backtest"},
	}, opts)
	if !res.Passed {
		t.Fatalf("execution should use judged turn tools: %s", res.Detail)
	}
}
