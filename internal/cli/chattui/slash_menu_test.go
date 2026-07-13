package chattui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func TestSlashMenuOpen(t *testing.T) {
	m := Model{
		input: textinput.New(),
	}
	m.input.SetValue("/")
	if !m.slashMenuOpen() {
		t.Fatal("expected slash menu open for /")
	}
	m.input.SetValue("hello")
	if m.slashMenuOpen() {
		t.Fatal("expected slash menu closed for plain text")
	}
}

func TestRenderSlashMenuShowsMatches(t *testing.T) {
	matches := chatui.MatchSlashCommands("/h")
	out := renderSlashMenu(matches, 0, 100)
	if !strings.Contains(out, "/help") {
		t.Fatalf("expected /help in menu: %q", out)
	}
}

func TestClampSlashPick(t *testing.T) {
	m := Model{input: textinput.New(), slashPick: 99}
	m.input.SetValue("/")
	m.clampSlashPick()
	if m.slashPick != len(chatui.SlashCommands)-1 {
		t.Fatalf("slashPick=%d want %d", m.slashPick, len(chatui.SlashCommands)-1)
	}
}
