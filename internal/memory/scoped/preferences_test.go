package scoped

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

func TestProfileBackendGetFileFallbackNoRecursion(t *testing.T) {
	home := t.TempDir()
	userID := "u1"
	ref := chatprompt.ProfileRef{Kind: chatprompt.ProfileUserDefault, Key: "u1"}
	if err := chatprompt.SaveProfile(home, userID, ref, "- test rule\n"); err != nil {
		t.Fatal(err)
	}
	backend := &ProfileBackend{Home: home, DB: &PreferencesStore{}}
	chatprompt.SetProfileBackend(backend)
	t.Cleanup(func() { chatprompt.SetProfileBackend(nil) })

	content, ok := backend.Get(context.Background(), userID, ref)
	if !ok || content == "" {
		t.Fatalf("expected file fallback content, got ok=%v", ok)
	}
}
