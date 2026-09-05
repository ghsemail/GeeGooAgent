package runtimeapi

import (
	"context"
	"testing"
	"time"
)

func TestClarifyHubQueuesFIFO(t *testing.T) {
	hub := newClarifyHub()
	sessionID := "sess-queue"
	firstDone := make(chan string, 1)
	secondDone := make(chan string, 1)

	go func() {
		answer, ok := hub.Wait(context.Background(), sessionID, "q1", []string{"a"}, nil)
		if ok {
			firstDone <- answer
		}
	}()
	time.Sleep(20 * time.Millisecond)

	go func() {
		answer, ok := hub.Wait(context.Background(), sessionID, "q2", []string{"b"}, nil)
		if ok {
			secondDone <- answer
		}
	}()
	time.Sleep(20 * time.Millisecond)

	if !hub.Answer(sessionID, "first", true) {
		t.Fatal("first answer failed")
	}
	if !hub.Answer(sessionID, "second", true) {
		t.Fatal("second answer failed")
	}

	select {
	case got := <-firstDone:
		if got != "first" {
			t.Fatalf("first waiter got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("first waiter timed out")
	}
	select {
	case got := <-secondDone:
		if got != "second" {
			t.Fatalf("second waiter got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("second waiter timed out")
	}
}

func TestClarifyHubPendingReturnsOldest(t *testing.T) {
	hub := newClarifyHub()
	sessionID := "sess-pending"
	done := make(chan struct{})

	go func() {
		_, _ = hub.Wait(context.Background(), sessionID, "first?", []string{"a"}, nil)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	go func() {
		_, _ = hub.Wait(context.Background(), sessionID, "second?", []string{"b"}, nil)
	}()
	time.Sleep(20 * time.Millisecond)

	p, ok := hub.Pending(sessionID)
	if !ok || p.Question != "first?" {
		t.Fatalf("pending=%+v ok=%v", p, ok)
	}
	hub.Answer(sessionID, "a", true)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("first waiter did not unblock")
	}
	hub.Answer(sessionID, "b", true)
}
