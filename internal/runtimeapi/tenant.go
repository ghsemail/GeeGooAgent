package runtimeapi

import (
	"net/http"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func resolveUserID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := strings.TrimSpace(r.Header.Get("X-User-Id")); v != "" {
		return v
	}
	return ""
}

func resolveUserIDWithFallback(r *http.Request, bodyUserID string) string {
	if v := resolveUserID(r); v != "" {
		return v
	}
	return strings.TrimSpace(bodyUserID)
}

func bindSessionUser(chat *chatsession.ChatSession, userID string) {
	chatsession.SetUserID(chat, userID)
}

func enforceSessionAccess(w http.ResponseWriter, chat *chatsession.ChatSession, userID string) bool {
	if chatsession.EnforceAccess(chat, userID) {
		return true
	}
	writeError(w, http.StatusForbidden, "session access denied")
	return false
}

func listSessionsForUser(store chatsession.SessionStore, userID string) ([]chatsession.ChatSessionIndexEntry, error) {
	entries, err := store.ListIndexedSessions()
	if err != nil {
		return nil, err
	}
	return chatsession.FilterEntriesByUser(entries, userID), nil
}

func latestSessionIDForUser(store chatsession.SessionStore, userID string) (string, error) {
	entries, err := listSessionsForUser(store, userID)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}
	return entries[0].ID, nil
}
