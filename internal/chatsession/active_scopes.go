package chatsession

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory/scoped"
)

const (
	metadataContextProfilesKey = "context_profiles"
	metadataActiveScopesKey    = "active_scopes"
)

// ActiveScopesFromSession returns session-bound scope keys (market/stock/automation).
// Reads both active_scopes and legacy context_profiles metadata.
func ActiveScopesFromSession(session *ChatSession) []string {
	if session == nil || session.Metadata == nil {
		return nil
	}
	var parts []string
	if raw, ok := session.Metadata[metadataActiveScopesKey]; ok {
		parts = append(parts, parseStringList(raw)...)
	}
	if raw, ok := session.Metadata[metadataContextProfilesKey]; ok {
		parts = append(parts, parseStringList(raw)...)
	}
	return mergeScopeStrings(parts)
}

// SetActiveScopes stores canonical active scope keys on the session.
func SetActiveScopes(session *ChatSession, scopes []string) {
	if session == nil {
		return
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}
	clean := mergeScopeStrings(scopes)
	if len(clean) == 0 {
		delete(session.Metadata, metadataActiveScopesKey)
		delete(session.Metadata, metadataContextProfilesKey)
		return
	}
	session.Metadata[metadataActiveScopesKey] = clean
	// Legacy readers (API / older clients) still see context_profiles.
	session.Metadata[metadataContextProfilesKey] = append([]string(nil), clean...)
}

// ContextProfilesFromSession returns active profile refs (alias for ActiveScopesFromSession).
func ContextProfilesFromSession(session *ChatSession) []string {
	return ActiveScopesFromSession(session)
}

// SetContextProfiles stores active scopes (writes both metadata keys).
func SetContextProfiles(session *ChatSession, refs []string) {
	SetActiveScopes(session, refs)
}

// MergeContextProfiles combines scope lists (deduped, order preserved).
func MergeContextProfiles(existing, incoming []string) []string {
	return mergeScopeStrings(append(existing, incoming...))
}

func parseStringList(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func mergeScopeStrings(parts []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, r := range parts {
		s := scoped.NormalizeScope(strings.TrimSpace(r))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
