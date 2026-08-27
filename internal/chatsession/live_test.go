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

func TestLivePublisherClarifyResolvedClearsClarifyStatus(t *testing.T) {
	state := infra.NewStateStore(t.TempDir())
	pub := chatsession.NewLivePublisher(state, "chat-clarify")
	pub.Emit("turn_start", nil)
	pub.Emit("clarify", map[string]any{"question": "pick"})
	pub.Emit("clarify_resolved", map[string]any{"answer": "60m", "auto": false})
	live, err := chatsession.LoadLiveState(state, "chat-clarify")
	if err != nil {
		t.Fatal(err)
	}
	if live.Status != "tool" {
		t.Fatalf("status=%q want tool", live.Status)
	}
	if live.LastEvent != "clarify_resolved" {
		t.Fatalf("last_event=%q", live.LastEvent)
	}
}
