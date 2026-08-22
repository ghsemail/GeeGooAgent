package chatprompt

import (
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

const agentsFileName = "AGENTS.md"

// ProfileKind identifies a context profile dimension.
type ProfileKind string

const (
	ProfileGlobal       ProfileKind = "global"
	ProfileUserDefault  ProfileKind = "user_default"
	ProfileMarket       ProfileKind = "market"
	ProfileStock        ProfileKind = "stock"
	ProfileAutomation   ProfileKind = "automation"
)

// ProfileRef is one loadable AGENTS.md anchor (type + key).
type ProfileRef struct {
	Kind ProfileKind
	Key  string
}

// String returns canonical ref form, e.g. "stock:00700.HK".
func (r ProfileRef) String() string {
	if r.Kind == ProfileGlobal || r.Kind == ProfileUserDefault {
		if k := strings.TrimSpace(r.Key); k != "" {
			return string(r.Kind) + ":" + k
		}
		return string(r.Kind)
	}
	return string(r.Kind) + ":" + strings.TrimSpace(r.Key)
}

// ParseProfileRef parses "market:HK", "stock:00700.HK", "automation:abc123".
func ParseProfileRef(raw string) (ProfileRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ProfileRef{}, errProfileRefEmpty
	}
	if !strings.Contains(raw, ":") {
		switch ProfileKind(raw) {
		case ProfileGlobal, ProfileUserDefault:
			return ProfileRef{Kind: ProfileKind(raw)}, nil
		default:
			return ProfileRef{}, errProfileRefInvalid
		}
	}
	i := strings.Index(raw, ":")
	kind := ProfileKind(strings.TrimSpace(raw[:i]))
	key := strings.TrimSpace(raw[i+1:])
	switch kind {
	case ProfileMarket, ProfileStock, ProfileAutomation:
		if key == "" {
			return ProfileRef{}, errProfileRefInvalid
		}
		return ProfileRef{Kind: kind, Key: key}, nil
	case ProfileGlobal, ProfileUserDefault:
		return ProfileRef{Kind: kind, Key: key}, nil
	case ProfileKind("bot"):
		if key == "" {
			return ProfileRef{}, errProfileRefInvalid
		}
		return ProfileRef{Kind: ProfileAutomation, Key: key}, nil
	default:
		return ProfileRef{}, errProfileRefInvalid
	}
}

// AgentsPathForRef resolves the on-disk AGENTS.md path for a profile ref.
func AgentsPathForRef(home, userID string, ref ProfileRef) string {
	if strings.TrimSpace(home) == "" {
		home = config.Home()
	}
	uid := SanitizeUserID(userID)
	switch ref.Kind {
	case ProfileGlobal:
		return filepath.Join(home, agentsFileName)
	case ProfileUserDefault:
		if uid == "" {
			return filepath.Join(home, agentsFileName)
		}
		return filepath.Join(home, "tenants", uid, agentsFileName)
	case ProfileMarket:
		return filepath.Join(home, "tenants", uid, "markets", sanitizeProfileKey(ref.Key), agentsFileName)
	case ProfileStock:
		return filepath.Join(home, "tenants", uid, "stocks", sanitizeProfileKey(ref.Key), agentsFileName)
	case ProfileAutomation:
		return filepath.Join(home, "tenants", uid, "automations", sanitizeProfileKey(ref.Key), agentsFileName)
	default:
		return ""
	}
}

func sanitizeProfileKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "..", "_")
	key = strings.ReplaceAll(key, string(filepath.Separator), "_")
	return key
}

// ProfileKindPriority orders session profile merge (lower first).
func ProfileKindPriority(kind ProfileKind) int {
	switch kind {
	case ProfileMarket:
		return 1
	case ProfileStock:
		return 2
	case ProfileAutomation:
		return 3
	default:
		return 99
	}
}
