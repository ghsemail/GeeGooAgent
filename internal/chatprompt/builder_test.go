package chatprompt_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

func TestSystemBuilderMatchesLegacySystem(t *testing.T) {
	got := chatprompt.DefaultBuilder().Build()
	want := chatprompt.System()
	if got != want {
		t.Fatalf("builder output drifted from System()")
	}
	if !strings.Contains(got, "Tool 路由") || !strings.Contains(got, "记忆：") {
		t.Fatalf("missing sections: len=%d", len(got))
	}
}

func TestSystemBuilderStableAcrossCalls(t *testing.T) {
	a := chatprompt.DefaultBuilder().Build()
	b := chatprompt.DefaultBuilder().Build()
	if a != b {
		t.Fatal("system prompt not stable")
	}
}
