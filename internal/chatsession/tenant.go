package chatsession

import "strings"

const metadataUserIDKey = "user_id"

// UserIDFromSession returns the owning user id from session metadata or index entry.
func UserIDFromSession(session *ChatSession) string {
	if session == nil || session.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(stringField(session.Metadata, metadataUserIDKey))
}

// UserIDFromEntry returns user_id from a list index entry metadata.
func UserIDFromEntry(entry ChatSessionIndexEntry) string {
	if entry.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(stringField(entry.Metadata, metadataUserIDKey))
}

// SetUserID stamps session ownership (also persisted via Postgres user_id column on Save).
func SetUserID(session *ChatSession, userID string) {
	if session == nil {
		return
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}
	session.Metadata[metadataUserIDKey] = userID
}

// FilterEntriesByUser returns sessions owned by userID. When userID is empty, all entries are returned (ops mode).
func FilterEntriesByUser(entries []ChatSessionIndexEntry, userID string) []ChatSessionIndexEntry {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return entries
	}
	out := make([]ChatSessionIndexEntry, 0, len(entries))
	for _, e := range entries {
		if UserIDFromEntry(e) == userID {
			out = append(out, e)
		}
	}
	return out
}

// EnforceAccess ensures the caller may use the session. Empty owner sessions are claimed by the first authenticated user.
func EnforceAccess(session *ChatSession, userID string) bool {
	userID = strings.TrimSpace(userID)
	if session == nil {
		return false
	}
	if userID == "" {
		return true
	}
	owner := UserIDFromSession(session)
	if owner == "" {
		SetUserID(session, userID)
		return true
	}
	return owner == userID
}
