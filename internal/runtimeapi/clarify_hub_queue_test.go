package runtimeapi

import (
	"context"
	"testing"
	"time"
)

func TestClarifyHubQueuesParallelWaits(t *testing.T) {
	h := newClarifyHub()
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	go func() {
		ans, ok := h.Wait(context.Background(), "sess-1", "first", []string{"A"}, nil)
		if !ok || ans != "A" {
			t.Errorf("wait1 = %q ok=%v", ans, ok)
		}
		close(done1)
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		ans, ok := h.Wait(context.Background(), "sess-1", "second", []string{"B"}, nil)
		if !ok || ans != "B" {
			t.Errorf("wait2 = %q ok=%v", ans, ok)
		}
		close(done2)
	}()
	time.Sleep(20 * time.Millisecond)
	if !h.Answer("sess-1", "A", true) {
		t.Fatal("answer1 failed")
	}
	if !h.Answer("sess-1", "B", true) {
		t.Fatal("answer2 failed")
	}
	select {
	case <-done1:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting first clarify")
	}
	select {
	case <-done2:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting second clarify")
	}
}
