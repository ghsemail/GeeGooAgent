package markdownnorm

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

// NormalizeAssistantReply fixes glued markdown from streaming models before storage/UI.
func NormalizeAssistantReply(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	return chatui.PreprocessTerminalMarkdown(text)
}
