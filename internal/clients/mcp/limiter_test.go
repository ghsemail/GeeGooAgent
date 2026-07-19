package mcp

import (
	"context"
	"testing"
)

func TestConcurrencyLimiterBlocksAndReleases(t *testing.T) {
	t.Parallel()
	l := NewConcurrencyLimiter(1)
	if err := l.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	go func() {
		_ = l.Acquire(context.Background())
		close(acquired)
	}()
	select {
	case <-acquired:
		t.Fatal("second acquire should block")
	default:
	}
	l.Release()
	<-acquired
	l.Release()
}

func TestConcurrencyLimiterNilNoop(t *testing.T) {
	t.Parallel()
	var l *ConcurrencyLimiter
	if err := l.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	l.Release()
}
