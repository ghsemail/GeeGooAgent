package retrievalgate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// ShouldRetrieve asks a small model whether long-term memory is needed (Waku parity).
// On any error, fails open: retrieve with the original message as query.
func ShouldRetrieve(ctx context.Context, provider llm.Provider, policy llm.Policy, message string) Decision {
	message = strings.TrimSpace(message)
	if message == "" {
		return Decision{Retrieve: false, Reason: "empty message"}
	}
	if provider == nil {
		return heuristicFallback(message)
	}
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
		return Decision{Retrieve: true, Query: message, Reason: "gate failed open (chat error)"}
	}
	text := strings.TrimSpace(resp.Content)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return Decision{Retrieve: true, Query: message, Reason: "gate returned no JSON — failing open"}
	}
	var parsed gateJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return Decision{Retrieve: true, Query: message, Reason: "gate parse error — failing open"}
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

// heuristicFallback is used when no gate provider is configured (tests / offline).
func heuristicFallback(message string) Decision {
	lower := strings.ToLower(message)
	for _, kw := range []string{"之前", "上次", "记得", "还记得", "remember", "recall"} {
		if strings.Contains(lower, kw) {
			return Decision{Retrieve: true, Query: message, Reason: "heuristic: " + kw}
		}
	}
	return Decision{Retrieve: false, Reason: "heuristic: no memory cue"}
}
