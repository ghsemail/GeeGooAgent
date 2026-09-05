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
	waiters map[string][]*clarifyWaiter
}

func newClarifyHub() *ClarifyHub {
	return &ClarifyHub{waiters: map[string][]*clarifyWaiter{}}
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
	w := &clarifyWaiter{
		question: question,
		choices:  append([]string(nil), choices...),
		ch:       make(chan clarifyAnswer, 1),
	}
	h.mu.Lock()
	h.waiters[sessionID] = append(h.waiters[sessionID], w)
	h.mu.Unlock()
	if onPending != nil {
		onPending(PendingClarify{
			SessionID: sessionID,
			Question:  question,
			Choices:   append([]string(nil), choices...),
		})
	}
	select {
	case res := <-w.ch:
		return res.answer, res.ok
	case <-ctx.Done():
		h.removeWaiter(sessionID, w)
		return "", false
	}
}

func (h *ClarifyHub) removeWaiter(sessionID string, target *clarifyWaiter) {
	if h == nil || target == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	queue := h.waiters[sessionID]
	out := queue[:0]
	for _, w := range queue {
		if w != target {
			out = append(out, w)
		}
	}
	if len(out) == 0 {
		delete(h.waiters, sessionID)
		return
	}
	h.waiters[sessionID] = out
}

// Answer unblocks the oldest pending clarify for sessionID.
func (h *ClarifyHub) Answer(sessionID, answer string, ok bool) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	queue := h.waiters[sessionID]
	if len(queue) == 0 {
		h.mu.Unlock()
		return false
	}
	w := queue[0]
	queue = queue[1:]
	if len(queue) == 0 {
		delete(h.waiters, sessionID)
	} else {
		h.waiters[sessionID] = queue
	}
	h.mu.Unlock()
	w.ch <- clarifyAnswer{answer: answer, ok: ok}
	return true
}

// Pending returns the oldest pending clarify for sessionID if any.
func (h *ClarifyHub) Pending(sessionID string) (PendingClarify, bool) {
	if h == nil {
		return PendingClarify{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	queue := h.waiters[sessionID]
	if len(queue) == 0 || queue[0] == nil {
		return PendingClarify{}, false
	}
	w := queue[0]
	return PendingClarify{
		SessionID: sessionID,
		Question:  w.question,
		Choices:   append([]string(nil), w.choices...),
	}, true
}
