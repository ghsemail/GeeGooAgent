package agent

import (
	"testing"

	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestApplyDynamicFragmentsInjectsBeforeUser(t *testing.T) {
	t.Parallel()
	loop := NewLoop(nil, nil)
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "hello"})

	var applied []string
	loop.SetProgress(func(event string, data map[string]any) {
		if event != "context_fragment_applied" {
			return
		}
		if raw, ok := data["applied"].([]string); ok {
			applied = raw
		}
	})

	frags := []ctxfrag.Fragment{
		ctxfrag.RecallFragment("semantic facts", "- user prefers HK stocks"),
		ctxfrag.ProceduralSkillFragment("step 1: check price"),
	}
	loop.applyDynamicFragments(session, frags, nil)

	if len(session.Messages) != 3 {
		t.Fatalf("messages=%d want 3 (default system + dynamic + user)", len(session.Messages))
	}
	if session.Messages[0].Role != llm.RoleSystem {
		t.Fatalf("first role=%s", session.Messages[0].Role)
	}
	if session.Messages[1].Role != llm.RoleSystem {
		t.Fatalf("dynamic fragment role=%s", session.Messages[1].Role)
	}
	if session.Messages[2].Role != llm.RoleUser {
		t.Fatalf("user role=%s", session.Messages[2].Role)
	}
	if len(applied) == 0 {
		t.Fatal("expected context_fragment_applied event")
	}
}
