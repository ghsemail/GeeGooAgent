package eval

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestVerifyTurnPlanLiveUsesInitialPlanForClarifyIntent(t *testing.T) {
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"initial_turn_plan": map[string]any{
				"domain": "ambiguous", "mode": "clarify", "sop": false,
			},
			"last_turn_plan": map[string]any{
				"domain": "stock_analysis", "mode": "gather", "act": "quote_price", "sop": false,
			},
		},
	}
	opts := TurnPlanCaseOptions{
		ExpectDomain: "ambiguous",
		ExpectMode:   "clarify",
		ExpectSOP:    false,
	}.Normalize()
	res := VerifyTurnPlanLive(chat, opts)
	if !res.Passed {
		t.Fatalf("expected pass using initial plan, got %s", res.Detail)
	}
}
