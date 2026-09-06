package runtimeapi

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime/events"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ClarifyChoiceCard is one labeled option for Web/TUI clients.
type ClarifyChoiceCard struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

func normalizeClarifyChoiceList(choices []string) []string {
	out := make([]string, 0, len(choices))
	for _, c := range choices {
		s := strings.TrimSpace(c)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func clarifyChoiceCards(choices []string) []ClarifyChoiceCard {
	display := tools.ClarifyDisplayOptions(choices)
	if len(display) == 0 {
		return []ClarifyChoiceCard{}
	}
	cards := make([]ClarifyChoiceCard, 0, len(display))
	for i, text := range display {
		cards = append(cards, ClarifyChoiceCard{
			Label: tools.ClarifyChoiceLabel(i),
			Text:  text,
		})
	}
	return cards
}

// ClarifyProgressPayload is the SSE/status body Dock Chat reads for the option sheet.
// choices is never null (empty slice if open-ended). display_choices includes A/B labels
// plus「其他（自行输入）」. Nested data mirrors the same fields for clients that only
// unwrap ProgressPayload.data.
func ClarifyProgressPayload(sessionID, question string, choices []string) map[string]any {
	choices = normalizeClarifyChoiceList(choices)
	q := strings.TrimSpace(question)
	if q == "" {
		q = "请确认"
	}
	cards := clarifyChoiceCards(choices)
	inner := map[string]any{
		"session_id":      sessionID,
		"question":        q,
		"choices":         choices,
		"options":         choices,
		"display_choices": cards,
	}
	return map[string]any{
		"session_id":      sessionID,
		"question":        q,
		"choices":         choices,
		"options":         choices,
		"display_choices": cards,
		"item_type":       events.ItemClarifyPrompt,
		"event":           "clarify",
		"data":            inner,
	}
}
