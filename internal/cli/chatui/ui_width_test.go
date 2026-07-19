package chatui

import "testing"

func TestAssistantWrapWidth_CappedOnWideTerminal(t *testing.T) {
	if got := assistantWrapWidth(200); got != 196 {
		t.Fatalf("wide terminal: got %d want 196", got)
	}
}

func TestAssistantWrapWidth_ShrinksOnNarrowTerminal(t *testing.T) {
	if got := assistantWrapWidth(50); got != 46 {
		t.Fatalf("narrow terminal should shrink: got %d want 46", got)
	}
}

func TestAssistantBoxOuterWidth_FixedOnWideTerminal(t *testing.T) {
	if got := assistantBoxOuterWidth(200); got != assistantBoxInnerWidth+4 {
		t.Fatalf("outer width: got %d want %d", got, assistantBoxInnerWidth+4)
	}
}
