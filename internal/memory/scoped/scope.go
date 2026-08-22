package scoped

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
)

// ScopeUser is the tenant-wide scope key (maps from user_default profile).
const ScopeUser = "user"

// ScopeGlobal is deployment-wide preferences (optional).
const ScopeGlobal = "global"

// NormalizeScope canonicalizes scope strings (bot → automation, user_default → user).
func NormalizeScope(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ScopeUser
	}
	switch raw {
	case "user", "user_default":
		return ScopeUser
	case "global":
		return ScopeGlobal
	}
	if strings.HasPrefix(raw, "bot:") {
		return "automation:" + strings.TrimSpace(raw[4:])
	}
	if ref, err := chatprompt.ParseProfileRef(raw); err == nil {
		return ScopeFromRef(ref)
	}
	return raw
}

// ScopeFromRef maps a profile ref to a scoped memory key.
func ScopeFromRef(ref chatprompt.ProfileRef) string {
	switch ref.Kind {
	case chatprompt.ProfileGlobal:
		return ScopeGlobal
	case chatprompt.ProfileUserDefault:
		return ScopeUser
	case chatprompt.ProfileMarket:
		return "market:" + strings.TrimSpace(ref.Key)
	case chatprompt.ProfileStock:
		return "stock:" + strings.TrimSpace(ref.Key)
	case chatprompt.ProfileAutomation:
		return "automation:" + strings.TrimSpace(ref.Key)
	default:
		return ref.String()
	}
}

// RefFromScope parses a scope key back to a profile ref when possible.
func RefFromScope(scope string) (chatprompt.ProfileRef, bool) {
	scope = NormalizeScope(scope)
	switch scope {
	case ScopeUser:
		return chatprompt.ProfileRef{Kind: chatprompt.ProfileUserDefault}, true
	case ScopeGlobal:
		return chatprompt.ProfileRef{Kind: chatprompt.ProfileGlobal}, true
	}
	ref, err := chatprompt.ParseProfileRef(scope)
	if err != nil {
		return chatprompt.ProfileRef{}, false
	}
	return ref, true
}

// NormalizeScopeList dedupes and canonicalizes scope strings.
func NormalizeScopeList(raw []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range raw {
		n := NormalizeScope(s)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
