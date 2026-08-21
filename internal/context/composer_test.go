package context

import "testing"

func TestComposerDropOrder(t *testing.T) {
	t.Parallel()
	c := Composer{MaxBytes: 40}
	text, applied, dropped := c.Compose([]Fragment{
		StaticFragment{K: KindRecall, Text: "recall hit", Prio: 80},
		StaticFragment{K: KindToolResult, Text: "tool output long text here", Prio: 60},
		StaticFragment{K: KindSystemRules, Text: "must keep rules", Prio: 10},
	})
	if text == "" {
		t.Fatal("expected some text")
	}
	if len(dropped) == 0 {
		t.Fatal("expected drop")
	}
	if dropped[0] != KindRecall {
		t.Fatalf("dropped=%v applied=%v", dropped, applied)
	}
}
