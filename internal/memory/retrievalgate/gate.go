package retrievalgate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const gatePrompt = `You are a retrieval gate for a personal assistant's long-term memory.
Given the user's message, decide if answering well requires the user's stored
memories (facts about people, projects, preferences, or past events).

Reply with ONLY this JSON, nothing else:
{"retrieve": true/false, "query": "<search keywords if true, else empty>", "reason": "<5 words>"}

General knowledge, math, small talk, or self-contained requests → false.
Anything referencing the user's life, people, plans, or history → true.

User message: %s`

// LLMTimeout bounds the auxiliary gate model so the chat UI cannot stall
// on “记忆门控” when the provider is slow.
const LLMTimeout = 1500 * time.Millisecond

// Decision is the parsed gate outcome.
type Decision struct {
	Retrieve bool
	Query    string
	Reason   string
}

type gateJSON struct {
	Retrieve bool   `json:"retrieve"`
	Query    string `json:"query"`
	Reason   string `json:"reason"`
}

// HasMemoryCue reports whether the utterance likely needs long-term recall.
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
// The auxiliary LLM runs only when the utterance has a memory cue, and is
// bounded by LLMTimeout. Errors and timeouts skip retrieval (fail closed)
// so the UI is not blocked on gate.
func ShouldRetrieve(ctx context.Context, provider llm.Provider, policy llm.Policy, message string) Decision {
	message = strings.TrimSpace(message)
	if message == "" {
		return Decision{Retrieve: false, Reason: "empty message"}
	}
	if !HasMemoryCue(message) {
		return heuristicFallback(message)
	}
	if provider == nil {
		return heuristicFallback(message)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, LLMTimeout)
	defer cancel()
	msgs := []llm.Message{{Role: llm.RoleUser, Content: fmt.Sprintf(gatePrompt, message)}}
	temp := 0.2
	maxTok := 600
	if policy != nil {
		dec := policy.Decide(llm.Request{Kind: llm.TaskCompress})
		if dec.Temperature > 0 {
			temp = dec.Temperature
		}
		if dec.MaxTokens > 0 && dec.MaxTokens < maxTok {
			maxTok = dec.MaxTokens
		}
	}
	resp, err := provider.Chat(ctx, msgs, nil, temp, maxTok)
	if err != nil {
		return Decision{Retrieve: false, Reason: "gate skip (chat error/timeout)"}
	}
	text := strings.TrimSpace(resp.Content)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return Decision{Retrieve: false, Reason: "gate skip (no JSON)"}
	}
	var parsed gateJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return Decision{Retrieve: false, Reason: "gate skip (parse error)"}
	}
	query := strings.TrimSpace(parsed.Query)
	if parsed.Retrieve && query == "" {
		query = message
	}
	return Decision{
		Retrieve: parsed.Retrieve,
		Query:    query,
		Reason:   strings.TrimSpace(parsed.Reason),
	}
}

// heuristicFallback is used when no gate provider is configured, or the
// utterance has no memory cue (tests / offline / most chat turns).
func heuristicFallback(message string) Decision {
	lower := strings.ToLower(message)
	for _, kw := range memoryCueTokens {
		if strings.Contains(lower, kw) {
			return Decision{Retrieve: true, Query: message, Reason: "heuristic: " + kw}
		}
	}
	return Decision{Retrieve: false, Reason: "heuristic: no memory cue"}
}
