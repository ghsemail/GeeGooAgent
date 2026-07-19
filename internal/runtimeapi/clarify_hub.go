package runtimeapi

import (
	"context"
	"sync"
)

type clarifyAnswer struct {
	answer string
	ok     bool
}

type clarifyWaiter struct {
	question string
	choices  []string
	ch       chan clarifyAnswer
}

// ClarifyHub blocks agent clarify tool calls until Answer is submitted.
type ClarifyHub struct {
	mu      sync.Mutex
	waiters map[string]*clarifyWaiter
}

func newClarifyHub() *ClarifyHub {
	return &ClarifyHub{waiters: map[string]*clarifyWaiter{}}
}

// Pending describes an in-flight clarify prompt.
type PendingClarify struct {
	SessionID string
	Question  string
	Choices   []string
}

// Wait blocks until Answer or ctx is cancelled.
func (h *ClarifyHub) Wait(ctx context.Context, sessionID, question string, choices []string, onPending func(PendingClarify)) (string, bool) {
	if h == nil {
		return "", false
	}
	ch := make(chan clarifyAnswer, 1)
	h.mu.Lock()
	h.waiters[sessionID] = &clarifyWaiter{
		question: question,
		choices:  append([]string(nil), choices...),
		ch:       ch,
	}
	h.mu.Unlock()
	if onPending != nil {
		onPending(PendingClarify{
			SessionID: sessionID,
			Question:  question,
			Choices:   append([]string(nil), choices...),
		})
	}
	select {
	case res := <-ch:
		return res.answer, res.ok
	case <-ctx.Done():
		h.mu.Lock()
		delete(h.waiters, sessionID)
		h.mu.Unlock()
		return "", false
	}
}

// Answer unblocks a pending clarify for sessionID.
func (h *ClarifyHub) Answer(sessionID, answer string, ok bool) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	w := h.waiters[sessionID]
	delete(h.waiters, sessionID)
	h.mu.Unlock()
	if w == nil {
		return false
	}
	w.ch <- clarifyAnswer{answer: answer, ok: ok}
	return true
}

// Pending returns the current clarify for sessionID if any.
func (h *ClarifyHub) Pending(sessionID string) (PendingClarify, bool) {
	if h == nil {
		return PendingClarify{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	w, ok := h.waiters[sessionID]
	if !ok || w == nil {
		return PendingClarify{}, false
	}
	return PendingClarify{
		SessionID: sessionID,
		Question:  w.question,
		Choices:   append([]string(nil), w.choices...),
	}, true
}
