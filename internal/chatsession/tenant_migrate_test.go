package chatsession_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func TestStampOrphanSessions(t *testing.T) {
	root := t.TempDir()
	store := chatsession.NewChatSessionStore(infra.NewStateStore(filepath.Join(root, "state")))
	s1, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	chatsession.SetUserID(s2, "already")
	if err := store.Save(s2); err != nil {
		t.Fatal(err)
	}
	n, err := chatsession.StampOrphanSessions(store, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("stamped=%d want 1", n)
	}
	loaded, _ := store.Load(s1.ID)
	if chatsession.UserIDFromSession(loaded) != "u1" {
		t.Fatalf("owner=%q", chatsession.UserIDFromSession(loaded))
	}
}
