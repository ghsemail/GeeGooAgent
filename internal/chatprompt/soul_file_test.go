package chatprompt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

func TestLoadSoulFromHomeMissingFileUsesDefault(t *testing.T) {
	dir := t.TempDir()
	got := chatprompt.LoadSoulFromHome(dir)
	want := chatprompt.DefaultSoul()
	if got != want {
		t.Fatalf("expected default soul, got len=%d", len(got))
	}
}

func TestSaveAndLoadSoulFromHome(t *testing.T) {
	dir := t.TempDir()
	custom := "You are a test agent.\n"
	if err := chatprompt.SaveSoulToHome(dir, custom); err != nil {
		t.Fatal(err)
	}
	got := chatprompt.LoadSoulFromHome(dir)
	if !strings.Contains(got, "test agent") {
		t.Fatalf("unexpected soul: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "SOUL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestSaveSoulRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := chatprompt.SaveSoulToHome(dir, "   "); err == nil {
		t.Fatal("expected empty soul error")
	}
}
