package runtimeapi

import (
	"context"
	"testing"
	"time"
)

func TestClarifyHubAnswer(t *testing.T) {
	h := newClarifyHub()
	done := make(chan struct{})
	go func() {
		ans, ok := h.Wait(context.Background(), "sess-1", "pick one", []string{"A", "B"}, nil)
		if !ok || ans != "A" {
			t.Errorf("wait = %q ok=%v", ans, ok)
		}
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	if !h.Answer("sess-1", "A", true) {
		t.Fatal("answer failed")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestClarifyHubOnPendingSeesWaiter(t *testing.T) {
	h := newClarifyHub()
	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_, _ = h.Wait(context.Background(), "sess-2", "选标的", []string{"00700 腾讯"}, func(p PendingClarify) {
			if _, ok := h.Pending(p.SessionID); !ok {
				t.Error("waiter missing during onPending")
			}
			if !h.Answer(p.SessionID, p.Choices[0], true) {
				t.Error("answer during onPending failed")
			}
			close(ready)
		})
		close(done)
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("onPending not called")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("wait did not unblock")
	}
}
