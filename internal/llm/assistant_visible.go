package llm

import (
	"regexp"
	"strings"
)

var (
	thinkBlockRE    = regexp.MustCompile(`(?is)<(?:redacted_thinking|think)>(.*?)</(?:redacted_thinking|think)>`)
	unclosedThinkRE = regexp.MustCompile(`(?is)<(?:redacted_thinking|think)>[\s\S]*$`)
)

// VisibleAssistantContent returns user-facing text from an LLM response, stripping inline thinking tags.
func VisibleAssistantContent(content, reasoning string) string {
	visible := PublicAnswerContent(content)
	if visible != "" {
		return visible
	}
	return strings.TrimSpace(reasoning)
}

// PublicAnswerContent is the assistant answer channel only (never reasoning_content).
func PublicAnswerContent(content string) string {
	visible := strings.TrimSpace(thinkBlockRE.ReplaceAllString(content, ""))
	return strings.TrimSpace(unclosedThinkRE.ReplaceAllString(visible, ""))
}
