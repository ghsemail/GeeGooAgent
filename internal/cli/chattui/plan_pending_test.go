package chattui

import "testing"

func TestFormatPlanTools(t *testing.T) {
	t.Parallel()
	if got := formatPlanTools(nil); got != "（写操作）" {
		t.Fatalf("empty: %q", got)
	}
	if got := formatPlanTools([]string{"create_dca_bot", "update_dca_bot"}); got != "create_dca_bot, update_dca_bot" {
		t.Fatalf("joined: %q", got)
	}
}

func TestToolNamesFromAny(t *testing.T) {
	t.Parallel()
	if got := toolNamesFromAny([]string{"a", "b"}); len(got) != 2 {
		t.Fatalf("[]string: %v", got)
	}
	if got := toolNamesFromAny([]any{"x", 1, "y"}); len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("[]any: %v", got)
	}
}
