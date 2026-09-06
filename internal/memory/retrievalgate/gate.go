package retrievalgate

import (
	"context"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// Decision is the parsed gate outcome.
type Decision struct {
	Retrieve bool
	Query    string
	Reason   string
}

// HasMemoryCue reports whether the utterance explicitly references past context.
// Retrieval itself does not depend on this; Cursor-style recall runs every turn.
func HasMemoryCue(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	for _, kw := range memoryCueTokens {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

var memoryCueTokens = []string{"之前", "上次", "记得", "还记得", "remember", "recall"}

// ShouldRetrieve decides whether long-term memory is needed.
// Every non-empty user turn retrieves with the utterance as the FTS query.
// No auxiliary LLM: keyword gates and helper-model round-trips hid memories
// that Cursor-style agents keep in context by default.
func ShouldRetrieve(ctx context.Context, provider llm.Provider, policy llm.Policy, message string) Decision {
	_ = ctx
	_ = provider
	_ = policy
	message = strings.TrimSpace(message)
	if message == "" {
		return Decision{Retrieve: false, Reason: "empty message"}
	}
	reason := "always retrieve"
	if HasMemoryCue(message) {
		reason = "always retrieve (explicit cue)"
	}
	return Decision{Retrieve: true, Query: message, Reason: reason}
}
