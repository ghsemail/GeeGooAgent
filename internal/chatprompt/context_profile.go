package chatprompt

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	errProfileRefEmpty   = errors.New("context profile ref empty")
	errProfileRefInvalid = errors.New("context profile ref invalid")
)

// ErrProfileRefEmpty reports an empty profile ref string.
func ErrProfileRefEmpty() error { return errProfileRefEmpty }

// ErrProfileRefInvalid reports a malformed profile ref.
func ErrProfileRefInvalid() error { return errProfileRefInvalid }

// AgentsMaxBytes is the maximum AGENTS.md size accepted from dashboard edits.
const AgentsMaxBytes = 32 * 1024

// ProfileLimits bounds merged profile content per turn.
type ProfileLimits struct {
	MaxMergedBytes        int
	MaxProfilesPerSession int
}

// DefaultProfileLimits returns production defaults from the P1-4 spec.
func DefaultProfileLimits() ProfileLimits {
	return ProfileLimits{MaxMergedBytes: 32768, MaxProfilesPerSession: 4}
}

// LoadedProfile is one AGENTS.md file on disk.
type LoadedProfile struct {
	Ref     ProfileRef `json:"ref"`
	Path    string     `json:"path"`
	Content string     `json:"content"`
	Bytes   int        `json:"bytes"`
	Missing bool       `json:"missing,omitempty"`
}

// MergeResult is the merged AGENTS block for system prompt injection.
type MergeResult struct {
	Text      string          `json:"text"`
	Profiles  []LoadedProfile `json:"profiles"`
	Truncated bool            `json:"truncated,omitempty"`
}

// LoadProfile reads profile content (DB backend first, then file).
func LoadProfile(home, userID string, ref ProfileRef) (LoadedProfile, bool) {
	return loadProfileMerged(home, userID, ref)
}

func loadProfileFile(home, userID string, ref ProfileRef) (LoadedProfile, bool) {
	path := AgentsPathForRef(home, userID, ref)
	out := LoadedProfile{Ref: ref, Path: path}
	if path == "" {
		out.Missing = true
		return out, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		out.Missing = true
		return out, false
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		out.Missing = true
		return out, false
	}
	out.Content = strings.TrimRight(string(b), "\n") + "\n"
	out.Bytes = len([]byte(out.Content))
	return out, true
}

// SaveProfile writes profile content (DB backend when configured, else file).
func SaveProfile(home, userID string, ref ProfileRef, content string) error {
	text := strings.TrimSpace(content)
	if text == "" {
		return errAgentsEmpty
	}
	if len([]byte(text)) > AgentsMaxBytes {
		return errAgentsTooLarge
	}
	return saveProfileMerged(home, userID, ref, text)
}

func saveProfileFile(home, userID string, ref ProfileRef, content string) error {
	path := AgentsPathForRef(home, userID, ref)
	if path == "" {
		return errProfileRefInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := strings.TrimRight(content, "\n") + "\n"
	return os.WriteFile(path, []byte(payload), 0o644)
}

var (
	errAgentsEmpty    = errors.New("AGENTS cannot be empty")
	errAgentsTooLarge = errors.New("AGENTS exceeds size limit")
)

// ErrAgentsEmpty reports an empty AGENTS save attempt.
func ErrAgentsEmpty() error { return errAgentsEmpty }

// ErrAgentsTooLarge reports AGENTS content over AgentsMaxBytes.
func ErrAgentsTooLarge() error { return errAgentsTooLarge }

// ResolveProfileRefs parses and dedupes session profile ref strings.
func ResolveProfileRefs(raw []string, limits ProfileLimits) ([]ProfileRef, error) {
	maxN := limits.MaxProfilesPerSession
	if maxN <= 0 {
		maxN = DefaultProfileLimits().MaxProfilesPerSession
	}
	seen := map[string]struct{}{}
	var out []ProfileRef
	for _, s := range raw {
		ref, err := ParseProfileRef(s)
		if err != nil {
			return nil, err
		}
		switch ref.Kind {
		case ProfileGlobal, ProfileUserDefault:
			continue
		}
		key := ref.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
		if len(out) >= maxN {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		pi, pj := ProfileKindPriority(out[i].Kind), ProfileKindPriority(out[j].Kind)
		if pi != pj {
			return pi < pj
		}
		return out[i].String() < out[j].String()
	})
	return out, nil
}

// MergeProfiles loads global, user_default, and session profiles into one AGENTS block.
func MergeProfiles(home, userID string, sessionRefs []string, limits ProfileLimits) MergeResult {
	maxBytes := limits.MaxMergedBytes
	if maxBytes <= 0 {
		maxBytes = DefaultProfileLimits().MaxMergedBytes
	}
	var profiles []LoadedProfile
	var parts []string

	add := func(ref ProfileRef) {
		if lp, ok := loadProfileMerged(home, userID, ref); ok {
			profiles = append(profiles, lp)
			parts = append(parts, formatProfileSection(ref, lp.Content))
		} else {
			profiles = append(profiles, lp)
		}
	}

	add(ProfileRef{Kind: ProfileGlobal})
	add(ProfileRef{Kind: ProfileUserDefault, Key: SanitizeUserID(userID)})
	if refs, err := ResolveProfileRefs(sessionRefs, limits); err == nil {
		for _, ref := range refs {
			add(ref)
		}
	}

	text := strings.TrimSpace(strings.Join(parts, "\n\n"))
	truncated := false
	if len([]byte(text)) > maxBytes {
		text = truncateUTF8(text, maxBytes)
		truncated = true
	}
	return MergeResult{Text: text, Profiles: profiles, Truncated: truncated}
}

func formatProfileSection(ref ProfileRef, content string) string {
	label := ref.String()
	return "[Context Profile: " + label + "]\n" + strings.TrimSpace(content)
}

func truncateUTF8(s string, maxBytes int) string {
	if len([]byte(s)) <= maxBytes {
		return s
	}
	b := []byte(s)
	if maxBytes > len(b) {
		maxBytes = len(b)
	}
	for maxBytes > 0 && b[maxBytes-1]&0xC0 == 0x80 {
		maxBytes--
	}
	return strings.TrimSpace(string(b[:maxBytes])) + "\n…"
}

// SystemForUserProfiles builds chat system prompt with layered AGENTS profiles.
func SystemForUserProfiles(home, userID string, sessionRefs []string, limits ProfileLimits) string {
	merge := MergeProfiles(home, userID, sessionRefs, limits)
	sections := []string{SoulForUser(userID)}
	if t := strings.TrimSpace(merge.Text); t != "" {
		sections = append(sections, t)
	}
	sections = append(sections, ToolRouting(), MemoryRules(), ServiceEndpoints())
	return SystemBuilder{Sections: sections}.Build()
}

// InspectProfiles returns load status for debugging (geegoo inspect / dashboard).
func InspectProfiles(home, userID string, sessionRefs []string, limits ProfileLimits) MergeResult {
	return MergeProfiles(home, userID, sessionRefs, limits)
}
