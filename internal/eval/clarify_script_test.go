package eval

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestDialogueFromLiveCaseSplitClarify(t *testing.T) {
	cases := []TurnPlanLiveCase{
		{
			ID: "ambiguous_bare_macd", Message: "这个MACD信号平时该怎么用比较好",
			ClarifyReply: "先问答，先不操作", ExpectMode: "clarify",
		},
		{
			ID: "backtest_colloquial", Message: "帮我回测一下中际旭创",
			ClarifyReply: "用SAR加MACD组合回测", ExpectMode: "execute",
		},
	}
	dialogue := dialogueFromLiveCase(cases[0])
	if len(dialogue) != 2 {
		t.Fatalf("clarify dialogue=%d want 2", len(dialogue))
	}
	if dialogue[0].OnClarify || dialogue[0].Judge {
		t.Fatalf("opening turn should not be on_clarify/judge: %+v", dialogue[0])
	}
	if !dialogue[1].OnClarify || !dialogue[1].Judge {
		t.Fatalf("follow-up turn should be on_clarify+judge: %+v", dialogue[1])
	}

	executeDialogue := dialogueFromLiveCase(cases[1])
	if len(executeDialogue) != 2 {
		t.Fatalf("execute dialogue=%d want 2", len(executeDialogue))
	}
}

func TestVerifyIntentUsesFirstTurnPlanForSplitClarify(t *testing.T) {
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_plan_trace": []chatsession.TurnPlanTraceEntry{
				{Turn: 1, Plan: chatsession.TurnPlanSnapshot{Domain: "ambiguous", Mode: "clarify", SOP: false}},
				{Turn: 2, Plan: chatsession.TurnPlanSnapshot{Domain: "chat", Mode: "talk", SOP: false}},
			},
			"last_turn_plan": map[string]any{"domain": "chat", "mode": "talk", "sop": false},
		},
	}
	opts := TurnPlanCaseOptions{
		TurnID: "ambiguous_bare_macd", ExpectDomain: "ambiguous", ExpectMode: "clarify",
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "这个MACD信号平时该怎么用比较好"},
			{Role: "user", Text: "先问答，先不操作", OnClarify: true, Judge: true},
		},
	}.Normalize()

	check := verifyIntent(chat, opts.intent(), opts)
	if !check.Passed {
		t.Fatalf("expected first-turn intent pass, got %s", check.Detail)
	}
}
