package eval

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestDialogueFromLiveCaseSplitClarify(t *testing.T) {
	cases := []TurnPlanLiveCase{
		{
			ID: "ambiguous_bare_macd",
			SetupMessages: []string{"腾讯股票最近的价格如何"},
			Message:       "有没有适合腾讯股价的MACD信号策略",
			ClarifyReply: "SAR信号搭配MACD直方图趋势", ExpectMode: "clarify",
		},
		{
			ID: "backtest_colloquial", Message: "帮我回测一下中际旭创",
			ClarifyReply: "用SAR加MACD组合回测", ExpectMode: "execute",
		},
	}
	dialogue := dialogueFromLiveCase(cases[0])
	if len(dialogue) != 2 {
		t.Fatalf("clarify-intent dialogue=%d want 2", len(dialogue))
	}
	if dialogue[0].OnClarify || dialogue[0].Judge {
		t.Fatalf("setup turn should not be on_clarify/judge: %+v", dialogue[0])
	}
	if dialogue[1].OnClarify || !dialogue[1].Judge {
		t.Fatalf("message turn should be judge: %+v", dialogue[1])
	}

	executeDialogue := dialogueFromLiveCase(cases[1])
	if len(executeDialogue) != 2 {
		t.Fatalf("execute dialogue=%d want 2", len(executeDialogue))
	}
}

func TestVerifyIntentUsesJudgedTurnPlanForMultiTurnClarify(t *testing.T) {
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_plan_trace": []chatsession.TurnPlanTraceEntry{
				{Turn: 1, Plan: chatsession.TurnPlanSnapshot{Domain: "stock_analysis", Mode: "gather", SOP: false}},
				{Turn: 2, Plan: chatsession.TurnPlanSnapshot{Domain: "ambiguous", Mode: "clarify", SOP: false}},
				{Turn: 3, Plan: chatsession.TurnPlanSnapshot{Domain: "chat", Mode: "talk", SOP: false}},
			},
			"last_turn_plan": map[string]any{"domain": "chat", "mode": "talk", "sop": false},
		},
	}
	opts := TurnPlanCaseOptions{
		TurnID: "ambiguous_bare_macd", ExpectDomain: "ambiguous", ExpectMode: "clarify",
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "腾讯股票最近的价格如何"},
			{Role: "user", Text: "有没有适合腾讯股价的MACD信号策略", Judge: true},
		},
	}.Normalize()

	check := verifyIntent(chat, opts.intent(), opts)
	if !check.Passed {
		t.Fatalf("expected judged-turn (turn 2) intent pass, got %s", check.Detail)
	}
}

func TestUsesFirstTurnIntentVerifySingleTurnOnly(t *testing.T) {
	single := TurnPlanCaseOptions{
		ExpectDomain: "ambiguous", ExpectMode: "clarify",
		Dialogue: []EvalDialogueTurn{{Role: "user", Text: "腾讯股价怎么样", Judge: true}},
	}.Normalize()
	if !UsesFirstTurnIntentVerify(single) {
		t.Fatal("expected first-turn verify for single-turn clarify")
	}
	multi := TurnPlanCaseOptions{
		ExpectDomain: "ambiguous", ExpectMode: "clarify",
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "腾讯股票最近的价格如何"},
			{Role: "user", Text: "有没有适合腾讯股价的MACD信号策略", Judge: true},
		},
	}.Normalize()
	if UsesFirstTurnIntentVerify(multi) {
		t.Fatal("multi-turn clarify should verify judged turn plan")
	}
}
