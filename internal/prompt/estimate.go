package prompt

import "github.com/ghsemail/GeeGooAgent/internal/llm"

// EstimateTokens rough-estimates prompt size as sum(len(content))/4 (+ tool call names).
func EstimateTokens(messages []llm.Message) int {
	n := 0
	for _, m := range messages {
		n += len(m.Content)
		for _, tc := range m.ToolCalls {
			n += len(tc.Name) + 16
		}
	}
	if n <= 0 {
		return 0
	}
	return (n + 3) / 4
}
