package runtimeapi

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func TestLoadSessionStatusIncludesPendingClarify(t *testing.T) {
	state := infra.NewStateStore(t.TempDir())
	store := chatsession.NewChatSessionStore(state)
	session, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(session); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(&app.App{Config: &config.AppConfig{}, State: state}, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_, _ = h.clarify.Wait(ctx, session.ID, "选周期", []string{"日线", "周线"}, nil)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		if _, ok := h.clarify.Pending(session.ID); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("clarify waiter was not registered")
		}
		time.Sleep(5 * time.Millisecond)
	}

	payload, err := h.loadSessionStatus(store, session.ID, "query")
	if err != nil {
		t.Fatal(err)
	}
	if payload == nil || payload.PendingClarify == nil {
		t.Fatal("missing pending_clarify")
	}
	if payload.PendingClarify.Question != "选周期" {
		t.Fatalf("question=%q", payload.PendingClarify.Question)
	}
	if len(payload.PendingClarify.Choices) != 2 {
		t.Fatalf("choices=%v", payload.PendingClarify.Choices)
	}
	if !payload.Busy {
		t.Fatal("expected busy while clarify waits")
	}
}
