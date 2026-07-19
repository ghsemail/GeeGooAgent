package chattui

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func syncSlotPlanFromRepl(slot *LiveSlot, repl *chatrepl.Repl) {
	if slot == nil || repl == nil || repl.Session == nil {
		return
	}
	plan := repl.Session.PendingPlan
	if plan == nil || len(plan.ToolCalls) == 0 {
		return
	}
	slot.PlanPending = true
	slot.PlanTools = toolNamesFromCalls(plan.ToolCalls)
}

func toolNamesFromCalls(calls []llm.ToolCall) []string {
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		if strings.TrimSpace(call.Name) == "" {
			continue
		}
		names = append(names, call.Name)
	}
	return names
}

func toolNamesFromAny(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func formatPlanTools(names []string) string {
	if len(names) == 0 {
		return "（写操作）"
	}
	return strings.Join(names, ", ")
}
