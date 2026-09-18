package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunnerEmitsTaskflowCard(t *testing.T) {
	var events []string
	r := testRunner(t)
	r.OnProgress = func(event string, _ map[string]any) {
		events = append(events, event)
	}
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "腾讯"})
	_, handled := r.RunTurn(context.Background(), session, "你挨个跑一下", tools.Context{}, 1)
	if !handled {
		t.Fatal("expected handled")
	}
	found := false
	for _, e := range events {
		if e == "workflow_card" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events=%v", events)
	}
}
