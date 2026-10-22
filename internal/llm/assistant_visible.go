package llm

import "strings"

// VisibleAssistantContent returns user-facing text from an LLM response, stripping inline thinking tags.
func VisibleAssistantContent(content, reasoning string) string {
	visible, _ := SplitInlineThinking(content)
	if visible != "" {
		return visible
	}
	return strings.TrimSpace(reasoning)
}

// PublicAnswerContent is the assistant answer channel only (never reasoning_content).
func PublicAnswerContent(content string) string {
	visible, _ := SplitInlineThinking(content)
	return visible
}
