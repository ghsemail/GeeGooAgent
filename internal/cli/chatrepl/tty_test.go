package chatrepl

import "testing"

func TestSlashPromptOptionsIncludeHighContrastColors(t *testing.T) {
	opts := slashPromptOptions()
	if len(opts) < 8 {
		t.Fatalf("expected color options, got %d", len(opts))
	}
}

func TestRestoreTTYNilSafe(t *testing.T) {
	restoreTTY(nil)
	restoreTTY(&ttyState{})
}
