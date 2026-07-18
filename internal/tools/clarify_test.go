package tools

import "testing"

func TestHandleClarifyRequiresInteractive(t *testing.T) {
	res := handleClarify(Context{Interactive: false}, map[string]any{"question": "pick one"})
	if res.Status != StatusSkip {
		t.Fatalf("status=%s", res.Status)
	}
}

func TestHandleClarifyReturnsAnswer(t *testing.T) {
	res := handleClarify(Context{
		Interactive: true,
		ClarifyFn: func(question string, choices []string) (string, bool) {
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
