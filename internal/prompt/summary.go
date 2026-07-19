package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const summarySystem = `You compress prior conversation turns into a structured brief for a stock-analysis agent.
Fill these sections (use Chinese or English to match the source):
## Goal
## Constraints & Preferences
## Progress
### Done
### In Progress
### Blocked
## Key Decisions
## Relevant Symbols / Reports
## Next Steps
## Critical Context
If an earlier summary is provided, UPDATE it instead of rewriting from scratch.`

// Summarizer produces a structured compaction summary.
type Summarizer interface {
	Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error)
}

// ProviderSummarizer wraps an llm.Provider for structured compaction summaries.
type ProviderSummarizer struct {
	Provider llm.Provider
	Policy   llm.Policy // optional; temperature from TaskCompress when set
}

func (p *ProviderSummarizer) Summarize(ctx context.Context, middle []llm.Message, previousSummary string, maxTokens int) (string, error) {
	if p == nil || p.Provider == nil {
		return "", fmt.Errorf("summarizer provider nil")
	}
	var b strings.Builder
	if previousSummary != "" {
		b.WriteString("Previous summary:\n")
		b.WriteString(previousSummary)
		b.WriteString("\n\n")
	}
	b.WriteString("Turns to compress:\n")
	for _, m := range middle {
		b.WriteString(string(m.Role))
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: summarySystem},
		{Role: llm.RoleUser, Content: b.String()},
	}
	temp := 0.2
	if p.Policy != nil {
		d := p.Policy.Decide(llm.Request{Kind: llm.TaskCompress})
		if d.Temperature > 0 {
			temp = d.Temperature
		}
	}
	resp, err := p.Provider.Chat(ctx, msgs, nil, temp, maxTokens)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(resp.Content)
	if text == "" {
		return "", fmt.Errorf("empty summary")
	}
	return text, nil
}
