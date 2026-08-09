package gateway

import (
	"context"
	"strings"
	"testing"
)

type fakeEditable struct {
	sends []OutboundText
	edits []string
	ids   int
}

func (f *fakeEditable) SendTextID(_ context.Context, msg OutboundText) (string, error) {
	f.sends = append(f.sends, msg)
	f.ids++
	return "om_" + strings.Repeat("x", f.ids), nil
}

func (f *fakeEditable) EditText(_ context.Context, messageID, text string) error {
	f.edits = append(f.edits, messageID+"|"+text)
	return nil
}

func TestProgressPublisherCreatesThenEdits(t *testing.T) {
	ed := &fakeEditable{}
	p := newProgressPublisher(context.Background(), ed, "chat1", "om_user")
	p.OnEvent("tool_start", map[string]any{"name": "search_code"})
	if len(ed.sends) != 1 {
		t.Fatalf("want 1 send, got %d", len(ed.sends))
	}
	if !strings.Contains(ed.sends[0].Text, "search_code") {
		t.Fatalf("send body=%q", ed.sends[0].Text)
	}
	p.OnEvent("tool_done", map[string]any{"name": "search_code", "status": "ok", "duration_ms": int64(10)})
	p.Flush()
	if len(ed.edits) < 1 {
		t.Fatalf("want edit, got sends=%d edits=%d", len(ed.sends), len(ed.edits))
	}
	if !strings.Contains(ed.edits[len(ed.edits)-1], "✅") {
		t.Fatalf("edit=%q", ed.edits[len(ed.edits)-1])
	}
}
