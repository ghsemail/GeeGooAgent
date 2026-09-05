package chatsession_test

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

func TestLivePublisherRoundTrip(t *testing.T) {
	state := infra.NewStateStore(t.TempDir())
	pub := chatsession.NewLivePublisher(state, "chat-test")
	if pub == nil {
		t.Fatal("nil publisher")
	}
	pub.Emit("turn_start", nil)
	pub.Emit("tool_start", map[string]any{"name": "search_code"})
	pub.EndTurn()

	live, err := chatsession.LoadLiveState(state, "chat-test")
	if err != nil {
		t.Fatal(err)
	}
	if live == nil {
		t.Fatal("nil live state")
	}
	if live.Busy {
		t.Fatal("expected idle after EndTurn")
	}
	if live.Status != "ready" {
		t.Fatalf("status=%q", live.Status)
	}
	if len(live.Events) < 2 {
		t.Fatalf("events=%d", len(live.Events))
	}
	if live.UpdatedAt.IsZero() {
		t.Fatal("missing updated_at")
	}
	_ = time.Now()
}

func TestLivePublisherClarifyOverridesGate(t *testing.T) {
	state := infra.NewStateStore(t.TempDir())
	pub := chatsession.NewLivePublisher(state, "chat-clarify")
	pub.Emit("turn_start", nil)
	pub.Emit("status", map[string]any{"phase": "gate", "message": "记忆门控"})
	pub.Emit("clarify", map[string]any{"question": "你是想做哪一件？", "choices": []string{"A", "B"}})
	live, err := chatsession.LoadLiveState(state, "chat-clarify")
	if err != nil {
		t.Fatal(err)
	}
	if live.Status != "clarify" {
		t.Fatalf("status=%q want clarify", live.Status)
	}
}
