package tools

import (
	"context"
	"testing"
)

func TestHandleClarifyRequiresCallback(t *testing.T) {
	res := handleClarify(Context{}, map[string]any{"question": "pick one"})
	if res.Status != StatusError {
		t.Fatalf("status=%s", res.Status)
	}
}

func TestHandleClarifyWorksWithoutInteractiveFlag(t *testing.T) {
	res := handleClarify(Context{
		ClarifyFn: func(context.Context, string, []string) (string, bool) { return "ok", true },
	}, map[string]any{"question": "pick one"})
	if res.Status != StatusOK {
		t.Fatalf("status=%s", res.Status)
	}
}

func TestHandleClarifyReturnsAnswer(t *testing.T) {
	res := handleClarify(Context{
		Interactive: true,
		ClarifyFn: func(_ context.Context, question string, choices []string) (string, bool) {
			if question != "which?" || len(choices) != 2 {
				t.Fatalf("q=%q choices=%v", question, choices)
			}
			return "B option", true
		},
	}, map[string]any{
		"question": "which?",
		"choices":  []any{"A option", "B option"},
	})
	if res.Status != StatusOK || res.Data["user_response"] != "B option" {
		t.Fatalf("res=%+v", res)
	}
}

func TestHandleClarifyHonorsWaitContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	resCh := make(chan Result, 1)
	go func() {
		resCh <- handleClarify(Context{
			Ctx: ctx,
			ClarifyFn: func(waitCtx context.Context, _ string, _ []string) (string, bool) {
				close(started)
				<-waitCtx.Done()
				return "", false
			},
		}, map[string]any{"question": "pick"})
	}()
	<-started
	cancel()
	res := <-resCh
	if res.Status != StatusSkip {
		t.Fatalf("status=%s summary=%s", res.Status, res.Summary)
	}
}

func TestNormalizeClarifyChoicesCapsAtFour(t *testing.T) {
	raw := []any{"1", "2", "3", "4", "5"}
	got := normalizeClarifyChoices(raw)
	if len(got) != 4 {
		t.Fatalf("got %v", got)
	}
}

func TestClarifyDisplayOptionsAppendsOther(t *testing.T) {
	opts := ClarifyDisplayOptions([]string{"a", "b"})
	if len(opts) != 3 || opts[2] != ClarifyOtherLabel {
		t.Fatalf("opts=%v", opts)
	}
}
