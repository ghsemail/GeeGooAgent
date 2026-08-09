package gateway

import (
	"context"
	"testing"
)

type fakeIndicatorAdapter struct {
	PlatformVal Platform
	Calls       []string
	SendTexts   []OutboundText
}

func (f *fakeIndicatorAdapter) Platform() Platform { return f.PlatformVal }
func (f *fakeIndicatorAdapter) Connect(context.Context, InboundHandler) error {
	return nil
}
func (f *fakeIndicatorAdapter) Disconnect(context.Context) error { return nil }
func (f *fakeIndicatorAdapter) SendText(_ context.Context, msg OutboundText) error {
	f.SendTexts = append(f.SendTexts, msg)
	return nil
}
func (f *fakeIndicatorAdapter) SendMedia(context.Context, string, string, []byte, string) error {
	return ErrNotImplemented{Feature: "fake"}
}
func (f *fakeIndicatorAdapter) Configured() bool { return true }
func (f *fakeIndicatorAdapter) Status() AdapterStatus {
	return AdapterStatus{Platform: f.PlatformVal, Configured: true}
}
func (f *fakeIndicatorAdapter) MarkProcessing(_ context.Context, messageID string) error {
	f.Calls = append(f.Calls, "processing:"+messageID)
	return nil
}
func (f *fakeIndicatorAdapter) ClearProcessing(_ context.Context, messageID string) error {
	f.Calls = append(f.Calls, "clear:"+messageID)
	return nil
}
func (f *fakeIndicatorAdapter) MarkFailed(_ context.Context, messageID string) error {
	f.Calls = append(f.Calls, "failed:"+messageID)
	return nil
}

func TestProcessingIndicatorLifecycle(t *testing.T) {
	fake := &fakeIndicatorAdapter{PlatformVal: PlatformFeishu}
	var ind ProcessingIndicator = fake
	_ = ind.MarkProcessing(context.Background(), "om_1")
	_ = ind.ClearProcessing(context.Background(), "om_1")
	_ = ind.MarkProcessing(context.Background(), "om_2")
	_ = ind.MarkFailed(context.Background(), "om_2")
	want := []string{"processing:om_1", "clear:om_1", "processing:om_2", "failed:om_2"}
	if len(fake.Calls) != len(want) {
		t.Fatalf("calls=%v", fake.Calls)
	}
	for i := range want {
		if fake.Calls[i] != want[i] {
			t.Fatalf("calls[%d]=%q want %q", i, fake.Calls[i], want[i])
		}
	}
}

func TestDedupCache(t *testing.T) {
	d := newDedupCache(2)
	if !d.TryAdd("a") {
		t.Fatal("first a")
	}
	if d.TryAdd("a") {
		t.Fatal("dup a")
	}
	if !d.TryAdd("b") || !d.TryAdd("c") {
		t.Fatal("b/c")
	}
	// a should be evicted
	if !d.TryAdd("a") {
		t.Fatal("a after eviction")
	}
}

func TestAuthorize(t *testing.T) {
	r := &Runner{Config: Config{
		AllowedUsers: map[Platform]map[string]struct{}{
			PlatformFeishu: {"ou_ok": {}},
		},
		AllowAll: map[Platform]bool{PlatformFeishu: false},
	}}
	if !r.authorize(InboundEvent{Platform: PlatformFeishu, UserID: "ou_ok"}) {
		t.Fatal("ou_ok")
	}
	if r.authorize(InboundEvent{Platform: PlatformFeishu, UserID: "ou_no"}) {
		t.Fatal("ou_no")
	}
	r.Config.AllowAll[PlatformFeishu] = true
	if !r.authorize(InboundEvent{Platform: PlatformFeishu, UserID: "ou_no"}) {
		t.Fatal("allow all")
	}
}
