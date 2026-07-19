package chatsession_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/lineage"
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

func TestChatSessionStoreIndexesMetadata(t *testing.T) {
	root := t.TempDir()
	store := infra.NewStateStore(filepath.Join(root, "state"))
	sessions := chatsession.NewChatSessionStore(store)

	session, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	session.Messages = append(session.Messages,
		llm.Message{Role: llm.RoleUser, Content: "Check 00700.HK price and capital flow"},
		llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{
				{ID: "c1", Name: "get_current_price", Arguments: map[string]any{"code": "00700.HK"}},
				{ID: "c2", Name: "get_capital_flow", Arguments: map[string]any{"symbol": "00700.HK"}},
			},
		},
		llm.Message{
			Role: llm.RoleTool, ToolCallID: "c1",
			Content: `{"status":"ok","summary":"00700.HK price=380.5","data":{"code":"00700.HK","price":380.5}}`,
		},
	)
	session.Status = "closed"

	if err := sessions.Save(session); err != nil {
		t.Fatal(err)
	}

	entries, err := sessions.ListIndexedSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("indexed entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.ID != session.ID {
		t.Fatalf("entry id = %q, want %q", entry.ID, session.ID)
	}
	if entry.Title != "Check 00700.HK price and capital flow" {
		t.Fatalf("entry title = %q", entry.Title)
	}
	if entry.Summary == "" {
		t.Fatal("expected summary")
	}
	if !containsString(entry.ToolNames, "get_current_price") || !containsString(entry.ToolNames, "get_capital_flow") {
		t.Fatalf("tool names = %#v", entry.ToolNames)
	}
	if !containsString(entry.Tags, "00700.HK") {
		t.Fatalf("tags = %#v", entry.Tags)
	}

	loaded, err := sessions.Load(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != entry.Title || loaded.Summary == "" || !containsString(loaded.ToolNames, "get_current_price") {
		t.Fatalf("loaded metadata not populated: %#v", loaded)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestChatSessionStoreLoadsLegacySession(t *testing.T) {
	root := t.TempDir()
	store := infra.NewStateStore(filepath.Join(root, "state"))
	sessions := chatsession.NewChatSessionStore(store)

	created := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	if err := store.Save("chat/legacy", map[string]any{
		"id": "legacy", "status": "closed",
		"created_at": created.Format(time.RFC3339), "updated_at": created.Format(time.RFC3339),
		"messages": []map[string]any{
			{"role": "system", "content": "system"},
			{"role": "user", "content": "Legacy 600519.SH price"},
			{"role": "assistant", "tool_calls": []map[string]any{{"id": "c1", "name": "get_current_price", "arguments": map[string]any{"code": "600519.SH"}}}},
		},
		"step_records": []map[string]any{}, "step_counter": 0,
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := sessions.Load("legacy")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Legacy 600519.SH price" {
		t.Fatalf("title = %q", loaded.Title)
	}
	if !containsString(loaded.ToolNames, "get_current_price") || !containsString(loaded.Tags, "600519.SH") {
		t.Fatalf("metadata not derived: tools=%#v tags=%#v", loaded.ToolNames, loaded.Tags)
	}
}

func TestChatSessionStoreRebuildsIndex(t *testing.T) {
	root := t.TempDir()
	stateRoot := filepath.Join(root, "state")
	store := infra.NewStateStore(stateRoot)
	sessions := chatsession.NewChatSessionStore(store)

	session, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	session.Messages = append(session.Messages,
		llm.Message{Role: llm.RoleUser, Content: "Find BYD 002594.SZ"},
		llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "c1", Name: "search_code", Arguments: map[string]any{"code": "002594.SZ"}}}},
	)
	if err := sessions.Save(session); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(stateRoot, "chat_index.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := sessions.ListIndexedSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != session.ID {
		t.Fatalf("rebuilt entries = %#v", entries)
	}
	if !containsString(entries[0].ToolNames, "search_code") || !containsString(entries[0].Tags, "002594.SZ") {
		t.Fatalf("rebuilt metadata: tools=%#v tags=%#v", entries[0].ToolNames, entries[0].Tags)
	}
}

func TestChatSessionLineageMetadataRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := infra.NewStateStore(filepath.Join(root, "state"))
	sessions := chatsession.NewChatSessionStore(store)
	session, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	session.SyncLineageFromRuntime(session.ID, session.ID, 2)
	if err := sessions.Save(session); err != nil {
		t.Fatal(err)
	}
	loaded, err := sessions.Load(session.ID)
	if err != nil || loaded == nil {
		t.Fatalf("load: %v %#v", err, loaded)
	}
	parent, rootID, gen := loaded.LineageFromMetadata()
	if parent != session.ID || rootID != session.ID || gen != 2 {
		t.Fatalf("lineage parent=%q root=%q gen=%d", parent, rootID, gen)
	}
}

func TestChatSessionLineageChainRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := infra.NewStateStore(filepath.Join(root, "state"))
	sessions := chatsession.NewChatSessionStore(store)
	session, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	session.SyncLineageChain([]lineage.Record{
		{Generation: 1, ParentID: session.ID, MsgsBefore: 10, MsgsAfter: 4, TokensBefore: 1000, TokensAfter: 400, SummaryHash: "abcd1234"},
	})
	if err := sessions.Save(session); err != nil {
		t.Fatal(err)
	}
	loaded, err := sessions.Load(session.ID)
	if err != nil || loaded == nil {
		t.Fatalf("load: %v", err)
	}
	chain := loaded.LineageChainFromMetadata()
	if len(chain) != 1 || chain[0].SummaryHash != "abcd1234" {
		t.Fatalf("chain=%+v", chain)
	}
}
