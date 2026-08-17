package chatsession

import "strings"

// ListPreview returns a one-line preview for session list UIs.
func ListPreview(entry ChatSessionIndexEntry) string {
	if s := strings.TrimSpace(entry.Summary); s != "" {
		return s
	}
	if t := strings.TrimSpace(entry.Title); t != "" && t != entry.ID {
		return t
	}
	return ""
}
