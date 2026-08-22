package context

import "strings"

// DefaultComposeMaxBytes is the default byte budget for dynamic context fragments.
const DefaultComposeMaxBytes = 32 * 1024

// RegisteredKinds returns all fragment kind identifiers (for inspect / debug).
func RegisteredKinds() []Kind {
	return []Kind{
		KindToolResult,
		KindRecall,
		KindProcedural,
		KindWorkingState,
		KindHookInject,
		KindSystemRules,
		KindBudgetReminder,
	}
}

// KindsToStrings converts kinds to strings for events and logs.
func KindsToStrings(ks []Kind) []string {
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		if k == "" {
			continue
		}
		out = append(out, string(k))
	}
	return out
}

// RecallFragment wraps retrieval-gate memory hits.
func RecallFragment(label, block string) Fragment {
	block = strings.TrimSpace(block)
	if block == "" {
		return StaticFragment{}
	}
	prefix := "Retrieved from long-term memory (FTS)."
	if label != "" {
		prefix = "Retrieved from " + label + ". Use only if relevant:"
	}
	return StaticFragment{
		K:    KindRecall,
		Text: prefix + "\n" + block,
		Prio: 20,
	}
}

// ProceduralSkillFragment wraps matched SKILL.md instructions.
func ProceduralSkillFragment(block string) Fragment {
	block = strings.TrimSpace(block)
	if block == "" {
		return StaticFragment{}
	}
	return StaticFragment{
		K:    KindProcedural,
		Text: "Relevant skill instructions (procedural memory). Follow only if applicable:\n" + block,
		Prio: 25,
	}
}

// RoundBudgetFragment is an ephemeral per-round budget nudge (not persisted).
func RoundBudgetFragment(text string) Fragment {
	text = strings.TrimSpace(text)
	if text == "" {
		return StaticFragment{}
	}
	return StaticFragment{
		K:    KindBudgetReminder,
		Text: text,
		Prio: 5,
	}
}

// HookInjectFragment reserves hook output injection (P2-4).
func HookInjectFragment(text string) Fragment {
	text = strings.TrimSpace(text)
	if text == "" {
		return StaticFragment{}
	}
	return StaticFragment{
		K:    KindHookInject,
		Text: text,
		Prio: 40,
	}
}

// WorkingStateFragment reserves workflow working-state injection.
func WorkingStateFragment(text string) Fragment {
	text = strings.TrimSpace(text)
	if text == "" {
		return StaticFragment{}
	}
	return StaticFragment{
		K:    KindWorkingState,
		Text: text,
		Prio: 35,
	}
}

// ToolResultFragment wraps a tool output block for fragment assembly.
func ToolResultFragment(text string) Fragment {
	text = strings.TrimSpace(text)
	if text == "" {
		return StaticFragment{}
	}
	return StaticFragment{
		K:    KindToolResult,
		Text: text,
		Prio: 60,
	}
}
