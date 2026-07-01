package runtime_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestNewUpstreamSessionPreservesHistoryAndPopsLastUser(t *testing.T) {
	session, lastUser := runtime.NewUpstreamSession([]runtime.UpstreamMessage{
		{Role: "system", Content: "custom system"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "查腾讯"},
	})
	if lastUser != "查腾讯" {
		t.Fatalf("lastUser=%q", lastUser)
	}
	msgs := session.LLMMessages()
	if len(msgs) != 3 {
		t.Fatalf("len=%d want 3 (last user popped)", len(msgs))
	}
	if msgs[0].Role != llm.RoleSystem || msgs[0].Content != "custom system" {
		t.Fatalf("system=%+v", msgs[0])
	}
	if msgs[len(msgs)-1].Role != llm.RoleAssistant {
		t.Fatalf("last=%+v", msgs[len(msgs)-1])
	}
}

func TestNewUpstreamSessionAddsDefaultSystemWhenHistoryStartsWithUser(t *testing.T) {
	session, lastUser := runtime.NewUpstreamSession([]runtime.UpstreamMessage{
		{Role: "user", Content: "hello"},
	})
	if lastUser != "hello" {
		t.Fatalf("lastUser=%q", lastUser)
	}
	msgs := session.LLMMessages()
	if len(msgs) != 1 {
		t.Fatalf("len=%d want default system only after popping last user", len(msgs))
	}
	if msgs[0].Role != llm.RoleSystem || msgs[0].Content != chatprompt.System() {
		t.Fatalf("system=%+v", msgs[0])
	}
}
