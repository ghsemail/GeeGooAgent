package app

import "testing"

func TestNewSessionIDIsUniqueWithinProcess(t *testing.T) {
	first := newSessionID()
	second := newSessionID()
	if first == second {
		t.Fatalf("expected unique session IDs, got %q twice", first)
	}
}
