package decision_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent/decision"
)

func TestShadowNeverTakeover(t *testing.T) {
	t.Parallel()
	var events []string
	c := &decision.Coordinator{
		Mode: decision.ModeShadow,
		Hook: decision.StubHook{ChoiceID: "hold", Confidence: 0.99},
	}
	q := decision.Question{
		ID: "pg1", Kind: decision.KindPlanGate, Step: 1,
		Choices: []decision.Choice{{ID: "hold"}, {ID: "execute"}},
	}
	_, takeover := c.Consult(context.Background(), q, func(event string, _ map[string]any) {
		events = append(events, event)
	})
	if takeover {
		t.Fatal("shadow must not takeover")
	}
	if len(events) != 1 || events[0] != "decision" {
		t.Fatalf("events=%v", events)
	}
}

func TestHookErrorFallsBackWithoutTakeover(t *testing.T) {
	t.Parallel()
	c := &decision.Coordinator{
		Mode: decision.ModeReplace,
		Hook: decision.StubHook{Err: errors.New("jev down")},
	}
	q := decision.Question{ID: "e1", Kind: decision.KindEndTurn, Choices: []decision.Choice{{ID: "reply"}}}
	ans, takeover := c.Consult(context.Background(), q, nil)
	if takeover {
		t.Fatal("error must not takeover")
	}
	if ans.Who != decision.WhoFallback {
		t.Fatalf("who=%s", ans.Who)
	}
}

func TestReplacePlanGateNeverTakeoverEvenHighConfidence(t *testing.T) {
	t.Parallel()
	c := &decision.Coordinator{
		Mode:          decision.ModeReplace,
		MinConfidence: 0.5,
		Hook:          decision.StubHook{ChoiceID: "execute", Confidence: 1},
	}
	q := decision.Question{
		ID: "pg2", Kind: decision.KindPlanGate,
		Choices: []decision.Choice{{ID: "hold"}, {ID: "execute"}},
	}
	_, takeover := c.Consult(context.Background(), q, nil)
	if takeover {
		t.Fatal("plan_gate must never takeover")
	}
}

func TestReplaceEndTurnCanTakeoverWhenConfident(t *testing.T) {
	t.Parallel()
	c := &decision.Coordinator{
		Mode:          decision.ModeReplace,
		MinConfidence: 0.8,
		Hook:          decision.StubHook{ChoiceID: "reply", Confidence: 0.95},
	}
	q := decision.Question{
		ID: "et1", Kind: decision.KindEndTurn,
		Choices: []decision.Choice{{ID: "reply"}},
	}
	_, takeover := c.Consult(context.Background(), q, nil)
	if !takeover {
		t.Fatal("expected replace takeover for end_turn")
	}
}
