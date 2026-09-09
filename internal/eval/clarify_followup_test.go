package eval

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestDialogueExecutionPlanSplitsClarifyTurns(t *testing.T) {
	opts := TurnPlanCaseOptions{
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我回测一下中际旭创"},
			{Role: "user", Text: "用SAR加MACD组合回测", OnClarify: true},
		},
	}.Normalize()

	regular, clarify := DialogueExecutionPlan(opts)
	if len(regular) != 1 || regular[0].Text != "帮我回测一下中际旭创" {
		t.Fatalf("regular=%v", regular)
	}
	if len(clarify) != 1 || clarify[0].Text != "用SAR加MACD组合回测" {
		t.Fatalf("clarify=%v", clarify)
	}
}

func TestNeedsClarifyFollowupWhenRequiredToolMissing(t *testing.T) {
	opts := TurnPlanCaseOptions{
		RequireTools: []string{"run_strategy_backtest"},
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我回测一下中际旭创"},
			{Role: "user", Text: "用SAR加MACD组合回测", OnClarify: true},
		},
	}.Normalize()

	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Tools: []string{"search_code"}},
			},
		},
	}
	if !NeedsClarifyFollowup(chat, opts) {
		t.Fatal("expected clarify follow-up")
	}

	chat.Metadata["turn_tools_trace"] = []chatsession.TurnToolsEntry{
		{Tools: []string{"search_code", "run_strategy_backtest"}},
	}
	if NeedsClarifyFollowup(chat, opts) {
		t.Fatal("expected no clarify follow-up once tool ran")
	}
}

func TestPickClarifyAnswerMatchesChoice(t *testing.T) {
	answer, ok := PickClarifyAnswer("请选择策略", []string{
		"SAR信号搭配MACD直方图趋势",
		"MACD金叉死叉",
	}, []string{"用SAR加MACD组合回测"})
	if !ok {
		t.Fatal("expected match")
	}
	if answer != "SAR信号搭配MACD直方图趋势" {
		t.Fatalf("answer=%q", answer)
	}
}

func TestBacktestColloquialCaseIncludesClarifyDialogue(t *testing.T) {
	var found bool
	for _, c := range IndividualTurnPlanEvalCases() {
		if c.ID != "turn_plan_backtest_colloquial" {
			continue
		}
		found = true
		regular, clarify := DialogueExecutionPlan(c.Options)
		if len(regular) != 1 || len(clarify) != 1 {
			t.Fatalf("regular=%d clarify=%d", len(regular), len(clarify))
		}
		if clarify[0].Text != "用SAR加MACD组合回测" {
			t.Fatalf("clarify=%q", clarify[0].Text)
		}
	}
	if !found {
		t.Fatal("missing turn_plan_backtest_colloquial")
	}
}
