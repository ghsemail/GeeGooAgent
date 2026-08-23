package gateway

import (
	"context"
	"sync"
)

type clarifyAnswer struct {
	text string
	ok   bool
}

type clarifyWaiter struct {
	ch chan clarifyAnswer
}

// ClarifyHub blocks IM clarify tool calls until the user sends the next message.
type ClarifyHub struct {
	mu      sync.Mutex
	waiters map[string]*clarifyWaiter
}

func NewClarifyHub() *ClarifyHub {
	return newClarifyHub()
}

func newClarifyHub() *ClarifyHub {
	return &ClarifyHub{waiters: map[string]*clarifyWaiter{}}
}

// Wait blocks until DeliverAnswer or ctx is cancelled.
func (h *ClarifyHub) Wait(ctx context.Context, sessionKey string) (string, bool) {
	if h == nil {
		return "", false
	}
	ch := make(chan clarifyAnswer, 1)
	h.mu.Lock()
	h.waiters[sessionKey] = &clarifyWaiter{ch: ch}
	h.mu.Unlock()

	select {
	case res := <-ch:
		return res.text, res.ok
	case <-ctx.Done():
		h.mu.Lock()
		delete(h.waiters, sessionKey)
		h.mu.Unlock()
		return "", false
	}
}

// DeliverAnswer routes an inbound IM message to a pending clarify waiter.
func (h *ClarifyHub) DeliverAnswer(sessionKey, text string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	w := h.waiters[sessionKey]
	delete(h.waiters, sessionKey)
	h.mu.Unlock()
	if w == nil {
		return false
	}
	w.ch <- clarifyAnswer{text: text, ok: true}
	return true
}

// Pending reports whether sessionKey is waiting for a clarify reply.
func (h *ClarifyHub) Pending(sessionKey string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.waiters[sessionKey]
	return ok
}
