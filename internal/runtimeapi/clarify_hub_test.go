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
