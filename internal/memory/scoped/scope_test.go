package scoped_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/memory/scoped"
)

func TestNormalizeScope(t *testing.T) {
	if scoped.NormalizeScope("user_default") != scoped.ScopeUser {
		t.Fatal("user_default")
	}
	if scoped.NormalizeScope("bot:abc") != "automation:abc" {
		t.Fatal("bot alias")
	}
	if scoped.NormalizeScope("stock:00700.HK") != "stock:00700.HK" {
		t.Fatal("stock")
	}
}

func TestScopeFromRef(t *testing.T) {
	got := scoped.ScopeFromRef(chatprompt.ProfileRef{Kind: chatprompt.ProfileAutomation, Key: "b1"})
	if got != "automation:b1" {
		t.Fatalf("got %s", got)
	}
}
