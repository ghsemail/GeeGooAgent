package events_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime/events"
)

func TestItemTypeForEvent(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"turn_start":               events.ItemUserMessage,
		"gate":                     events.ItemStatus,
		"tool_start":               events.ItemToolCall,
		"tool_end":                 events.ItemToolResult,
		"tool_done":                events.ItemToolResult,
		"budget_warning":           events.ItemBudgetWarning,
		"turn_end":                 events.ItemTurnComplete,
		"clarify":                  events.ItemClarifyPrompt,
		"clarify_plan":             events.ItemClarifyPrompt,
		"context_fragment_applied": events.ItemStatus,
	}
	for event, want := range cases {
		if got := events.ItemTypeForEvent(event); got != want {
			t.Fatalf("event %q: got %q want %q", event, got, want)
		}
	}
}
