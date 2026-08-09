package gateway_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
)

func TestSessionKey(t *testing.T) {
	got := gateway.SessionKey(gateway.PlatformFeishu, "oc_chat", "ou_user")
	want := "feishu:oc_chat:u:ou_user"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSessionMapRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m, err := gateway.NewSessionMap(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Put("feishu:oc:u:ou", "sess-1"); err != nil {
		t.Fatal(err)
	}
	m2, err := gateway.NewSessionMap(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := m2.Get("feishu:oc:u:ou")
	if !ok || id != "sess-1" {
		t.Fatalf("got %q ok=%v", id, ok)
	}
	if _, err := os.Stat(filepath.Join(dir, "gateway", "sessions.json")); err != nil {
		t.Fatal(err)
	}
}

func TestDedupViaRepeatedPut(t *testing.T) {
	// Ensure map Put is idempotent for same key.
	dir := t.TempDir()
	m, err := gateway.NewSessionMap(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Put("k", "a")
	_ = m.Put("k", "b")
	id, _ := m.Get("k")
	if id != "b" {
		t.Fatalf("want b got %s", id)
	}
}
