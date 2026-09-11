package chatsession

import "testing"

func TestIntentTurnPlanFromSessionPrefersInitialForClarify(t *testing.T) {
	chat := &ChatSession{
		Metadata: map[string]any{
			"initial_turn_plan": map[string]any{
				"domain": "ambiguous", "mode": "clarify", "sop": false,
			},
			"last_turn_plan": map[string]any{
				"domain": "stock_analysis", "mode": "gather", "act": "quote_price", "sop": false,
			},
		},
	}
	snap, ok := IntentTurnPlanFromSession(chat, "clarify")
	if !ok || snap.Domain != "ambiguous" || snap.Mode != "clarify" {
		t.Fatalf("got %+v ok=%v", snap, ok)
	}
	snap, ok = IntentTurnPlanFromSession(chat, "gather")
	if !ok || snap.Domain != "stock_analysis" {
		t.Fatalf("non-clarify expect last plan, got %+v", snap)
	}
}
