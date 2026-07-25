package consolidation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
)

const metadataConsolidatedPairs = "memory_consolidated_pairs"

const summarizerPrompt = `You distill a personal assistant's recent conversation into long-term memory.

From the exchanges below, extract:
1. durable facts about the user, their people, projects, or preferences —
   only things worth remembering in a month; skip chit-chat and one-offs.
2. one single-sentence episode summarizing what happened in this conversation.

Reply with ONLY this JSON:
{"facts": [{"subject": "<who/what>", "content": "<one sentence>"}], "episode": "<one sentence>"}

Exchanges:
%s`

// Result reports what consolidation wrote.
type Result struct {
	Facts   int
	Episode bool
}

// Distiller batches chat exchanges into semantic facts and episodic summaries.
type Distiller struct {
	Provider llm.Provider
	Policy   llm.Policy
	Facts    *facts.PostgresStore
	Episodic *episodic.PostgresStore
	EveryN   int
}

// MaybeConsolidate runs when enough new user-assistant pairs accumulated.
func (d *Distiller) MaybeConsolidate(ctx context.Context, session *chatsession.ChatSession) (Result, error) {
	var out Result
	if d == nil || d.Provider == nil || session == nil {
		return out, nil
	}
	everyN := d.EveryN
	if everyN <= 0 {
		everyN = 3
	}
	pairs := countPairs(session.Messages)
	last := consolidatedPairs(session)
	if pairs-last < everyN {
		return out, nil
	}
	log := formatExchangeLog(session.Messages, last)
	if strings.TrimSpace(log) == "" {
		return out, nil
	}
	distilled, err := d.distill(ctx, log)
	if err != nil {
		return out, err
	}
	userID := chatsession.UserIDFromSession(session)
	for _, fact := range distilled.Facts {
		subject := strings.TrimSpace(fact.Subject)
		content := strings.TrimSpace(fact.Content)
		if subject == "" || content == "" {
			continue
		}
		if d.Facts != nil {
			if err := d.Facts.Add(ctx, userID, subject, content, "consolidation"); err == nil {
				out.Facts++
			}
		}
	}
	if ep := strings.TrimSpace(distilled.Episode); ep != "" && d.Episodic != nil {
		ep = "[consolidated] " + ep
		if err := d.Episodic.Add(ctx, session.ID, userID, ep, time.Now().UTC()); err == nil {
			out.Episode = true
		}
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}
	session.Metadata[metadataConsolidatedPairs] = pairs
	return out, nil
}

type factRow struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

type distilledMemory struct {
	Facts   []factRow `json:"facts"`
	Episode string    `json:"episode"`
}

func (d *Distiller) distill(ctx context.Context, log string) (distilledMemory, error) {
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: fmt.Sprintf(summarizerPrompt, log)},
	}
	temp := 0.2
	maxTok := 600
	if d.Policy != nil {
		dec := d.Policy.Decide(llm.Request{Kind: llm.TaskCompress})
		if dec.Temperature > 0 {
			temp = dec.Temperature
		}
		if dec.MaxTokens > 0 {
			maxTok = dec.MaxTokens
		}
	}
	resp, err := d.Provider.Chat(ctx, msgs, nil, temp, maxTok)
	if err != nil {
		return distilledMemory{}, err
	}
	text := strings.TrimSpace(resp.Content)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return distilledMemory{}, fmt.Errorf("consolidation: no JSON in response")
	}
	var out distilledMemory
	if err := json.Unmarshal([]byte(text[start:end+1]), &out); err != nil {
		return distilledMemory{}, err
	}
	return out, nil
}

func countPairs(msgs []llm.Message) int {
	n := 0
	pendingUser := false
	for _, m := range msgs {
		switch m.Role {
		case llm.RoleUser:
			pendingUser = true
		case llm.RoleAssistant:
			if pendingUser {
				n++
				pendingUser = false
			}
		}
	}
	return n
}

func consolidatedPairs(session *chatsession.ChatSession) int {
	if session == nil || session.Metadata == nil {
		return 0
	}
	switch v := session.Metadata[metadataConsolidatedPairs].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func formatExchangeLog(msgs []llm.Message, skipPairs int) string {
	var b strings.Builder
	pairs := 0
	for i := 0; i < len(msgs); i++ {
		if msgs[i].Role != llm.RoleUser {
			continue
		}
		user := strings.TrimSpace(msgs[i].Content)
		assistant := ""
		for j := i + 1; j < len(msgs); j++ {
			if msgs[j].Role == llm.RoleAssistant {
				assistant = strings.TrimSpace(msgs[j].Content)
				break
			}
		}
		if user == "" && assistant == "" {
			continue
		}
		if pairs < skipPairs {
			pairs++
			continue
		}
		fmt.Fprintf(&b, "user: %s\n", user)
		fmt.Fprintf(&b, "assistant: %s\n", assistant)
		pairs++
	}
	return b.String()
}
