package mcp

import "context"

// ConcurrencyLimiter caps concurrent outbound HTTP calls (shared across MCP clients).
type ConcurrencyLimiter struct {
	sem chan struct{}
}

// NewConcurrencyLimiter creates a limiter. max<=0 disables limiting.
func NewConcurrencyLimiter(max int) *ConcurrencyLimiter {
	if max <= 0 {
		return nil
	}
	return &ConcurrencyLimiter{sem: make(chan struct{}, max)}
}

// Acquire blocks until a slot is available or ctx is cancelled.
func (l *ConcurrencyLimiter) Acquire(ctx context.Context) error {
	if l == nil {
		return nil
	}
	select {
	case l.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release returns a slot acquired by Acquire.
func (l *ConcurrencyLimiter) Release() {
	if l == nil {
		return
	}
	select {
	case <-l.sem:
	default:
	}
}
