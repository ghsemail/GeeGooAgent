package agent

import "strings"

// IsExecutionRetryUserContent reports internal loop retry hints that must not appear in chat UI.
func IsExecutionRetryUserContent(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "[execution_retry]")
}
