package chatsession

import "strings"

const metadataContextProfilesKey = "context_profiles"

// ContextProfilesFromSession returns active profile refs from session metadata.
func ContextProfilesFromSession(session *ChatSession) []string {
	if session == nil || session.Metadata == nil {
		return nil
	}
	raw, ok := session.Metadata[metadataContextProfilesKey]
	if !ok || raw == nil {
		return nil
	}
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

// SetContextProfiles stores active profile refs on the session.
func SetContextProfiles(session *ChatSession, refs []string) {
	if session == nil {
		return
	}
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}
	if len(refs) == 0 {
		delete(session.Metadata, metadataContextProfilesKey)
		return
	}
	clean := make([]string, 0, len(refs))
	for _, r := range refs {
		if s := strings.TrimSpace(r); s != "" {
			clean = append(clean, s)
		}
	}
	session.Metadata[metadataContextProfilesKey] = clean
}

// MergeContextProfiles combines existing session refs with new refs (deduped, order preserved).
func MergeContextProfiles(existing, incoming []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(list []string) {
		for _, r := range list {
			s := strings.TrimSpace(r)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	add(existing)
	add(incoming)
	return out
}
