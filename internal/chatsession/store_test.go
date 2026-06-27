package chatsession_test

import (
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestChatSessionStoreRoundTripAndRecall(t *testing.T) {
	root := t.TempDir()
	store := infra.NewStateStore(filepath.Join(root, "state"))
	sessions := chatsession.NewChatSessionStore(store)

	old, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	old.Messages = append(old.Messages,
		llm.Message{Role: llm.RoleUser, Content: "腾讯股价多少"},
		llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{
				{ID: "c1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}},
			},
		},
		llm.Message{
			Role: llm.RoleTool, ToolCallID: "c1",
			Content: `{"status":"ok","summary":"00700.HK price=380.5","data":{"code":"00700.HK","price":380.5}}`,
		},
	)
	old.Status = "closed"
	if err := sessions.Save(old); err != nil {
		t.Fatal(err)
	}

	current, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	hits, err := chatsession.SearchPastSessions(sessions, "腾讯 股价", current.ID, 5, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected recall hit")
	}
	if hits[0].SessionID != old.ID {
		t.Fatalf("session id = %q", hits[0].SessionID)
	}
}
