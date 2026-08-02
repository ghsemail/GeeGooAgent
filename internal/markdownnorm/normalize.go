package markdownnorm

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

// NormalizeForStorage fixes glued markdown before persisting or sending to Web/API clients.
func NormalizeForStorage(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	return chatui.PreprocessWebMarkdown(text)
}

// NormalizeForTerminalDisplay applies web fixes plus terminal table adaptation.
func NormalizeForTerminalDisplay(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	return chatui.PreprocessTerminalMarkdown(text)
}

// NormalizeAssistantReply is an alias for NormalizeForStorage (session / turn_end payloads).
func NormalizeAssistantReply(text string) string {
	return NormalizeForStorage(text)
}
