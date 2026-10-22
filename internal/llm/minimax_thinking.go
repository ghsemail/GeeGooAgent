package llm

import (
	"context"
	"strings"
)

// MiniMaxM3SupportsThinkingDisable reports models where thinking.type=disabled is effective.
// See https://platform.minimax.io/docs/api-reference/text-openai-api — MiniMax-M3 skips thinking when disabled;
// omitting thinking leaves adaptive thinking ON (30–100s latency).
func MiniMaxM3SupportsThinkingDisable(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if !strings.Contains(m, "minimax") {
		return false
	}
	if strings.Contains(m, "m3.1-flash") || strings.Contains(m, "m3.1") && strings.Contains(m, "flash") {
		return false
	}
	return strings.Contains(m, "minimax-m3") || strings.Contains(m, "-m3")
}

// ModelCatalogLikelyForcedReasoning marks catalog rows that cannot skip long reasoning for compose.
func ModelCatalogLikelyForcedReasoning(name string) bool {
	m := strings.ToLower(strings.TrimSpace(name))
	if m == "" {
		return false
	}
	if strings.Contains(m, "reasoner") || strings.Contains(m, "deepseek-r1") {
		return true
	}
	if strings.Contains(m, "m3.1-flash") {
		return true
	}
	return false
}

func applyThinkingRequestParams(body map[string]any, model string, ctx context.Context, providerThinkingOn bool, reasoningEffort string) {
	meta := CallMetaFrom(ctx)
	if meta.ThinkingOff {
		if MiniMaxM3SupportsThinkingDisable(model) {
			body["thinking"] = map[string]any{"type": "disabled"}
			body["reasoning_split"] = true
		}
		return
	}
	if providerThinkingOn {
		body["thinking"] = map[string]any{"type": "enabled"}
		if effort := strings.TrimSpace(reasoningEffort); effort != "" {
			body["reasoning_effort"] = effort
		}
	}
}
