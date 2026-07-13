package chatui

import "testing"

func TestMatchSlashCommandsRoot(t *testing.T) {
	matches := MatchSlashCommands("/")
	if len(matches) != len(SlashCommands) {
		t.Fatalf("expected %d matches for /, got %d", len(SlashCommands), len(matches))
	}
}

func TestMatchSlashCommandsPrefix(t *testing.T) {
	matches := MatchSlashCommands("/hel")
	if len(matches) != 1 || matches[0].Command != "/help" {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}

func TestMatchSlashCommandsNoMatch(t *testing.T) {
	if got := MatchSlashCommands("hello"); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := MatchSlashCommands("/zzz"); len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
}
