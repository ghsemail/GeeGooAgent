package procedural

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
)

// SkillInjectionMode controls how Chat loads procedural skills each turn.
type SkillInjectionMode string

const (
	SkillInjectionKeyword SkillInjectionMode = "keyword"
	SkillInjectionCatalog SkillInjectionMode = "catalog"
	SkillInjectionOff     SkillInjectionMode = "off"
)

// ParseSkillInjectionMode normalizes config; unknown values default to catalog.
func ParseSkillInjectionMode(raw string) SkillInjectionMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "keyword", "match":
		return SkillInjectionKeyword
	case "off", "none", "disabled":
		return SkillInjectionOff
	case "catalog", "":
		return SkillInjectionCatalog
	default:
		return SkillInjectionCatalog
	}
}

// scheduledChatSkills are cron/run-only L5 skills excluded from the default Chat catalog.
var scheduledChatSkills = map[string]struct{}{
	"premarket_market": {},
	"premarket_stock":  {},
	"intraday_stock":   {},
	"postmarket_stock": {},
}

// InChatCatalog reports whether a skill should appear in the injected Chat catalog.
func InChatCatalog(sk Skill, policy SkillsPolicy) bool {
	kind := ClassifyPath(sk.Path)
	if !InjectInChat(kind) || !policy.Enabled(sk.Name, kind) {
		return false
	}
	name := normalizeSkillName(sk.Name)
	if _, skip := scheduledChatSkills[name]; skip {
		return false
	}
	if isScheduledOnlyPath(sk.Path) {
		return false
	}
	return true
}

func isScheduledOnlyPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	for _, seg := range []string{
		"/skills/premarket_market/", "/skills/premarket_stock/",
		"/skills/intraday_stock/", "/skills/postmarket_stock/",
	} {
		if strings.Contains(p, seg) {
			return true
		}
	}
	return false
}

// CatalogEntry is one skill row in the Chat catalog (id + short description).
type CatalogEntry struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Kind        Kind   `json:"kind"`
}

// ChatCatalog returns Chat-usable skills for catalog injection (sorted by id).
func (l *Loader) ChatCatalog() []CatalogEntry {
	if l == nil {
		return nil
	}
	l.maybeRefresh()
	policy := l.Policy()
	var out []CatalogEntry
	for _, sk := range l.skills {
		if !InChatCatalog(sk, policy) {
			continue
		}
		desc := strings.TrimSpace(sk.Description)
		if desc == "" {
			desc = strings.TrimSpace(sk.Body)
			if len(desc) > 160 {
				desc = truncateRunes(desc, 160)
			}
		}
		out = append(out, CatalogEntry{
			ID:          sk.Name,
			Description: desc,
			Kind:        ClassifyPath(sk.Path),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// CatalogEntriesForLockedSkill narrows the injected catalog to one Monday locked skill id.
func CatalogEntriesForLockedSkill(entries []CatalogEntry, lockedID string) []CatalogEntry {
	lockedID = strings.TrimSpace(lockedID)
	if lockedID == "" || len(entries) == 0 {
		return entries
	}
	var out []CatalogEntry
	for _, e := range entries {
		if e.ID == lockedID {
			out = append(out, e)
		}
	}
	return out
}

// FormatCatalog renders the catalog block for system context.
func FormatCatalog(entries []CatalogEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Available skills (call skill_view with skill id to load full SKILL.md before following steps):\n")
	for _, e := range entries {
		b.WriteString("- **")
		b.WriteString(e.ID)
		b.WriteString("**: ")
		b.WriteString(strings.TrimSpace(e.Description))
		b.WriteByte('\n')
	}
	return b.String()
}

// ReadSkillFile loads SKILL.md body or a sibling file under the skill directory.
func (l *Loader) ReadSkillFile(name, relFile string) (Skill, string, error) {
	sk, ok := l.FindByName(name)
	if !ok {
		return Skill{}, "", ErrSkillNotFound
	}
	relFile = strings.TrimSpace(relFile)
	if relFile == "" || relFile == "SKILL.md" {
		body := strings.TrimSpace(sk.Body)
		if body == "" {
			body = sk.Description
		}
		return sk, body, nil
	}
	relFile = filepath.Clean(relFile)
	if relFile == ".." || strings.HasPrefix(relFile, ".."+string(filepath.Separator)) {
		return Skill{}, "", ErrSkillFileOutside
	}
	dir := filepath.Dir(sk.Path)
	target := filepath.Join(dir, relFile)
	raw, err := readFileUnder(dir, target)
	if err != nil {
		return sk, "", err
	}
	return sk, string(raw), nil
}

var (
	ErrSkillNotFound    = errors.New("skill not found")
	ErrSkillFileOutside = errors.New("file must stay under skill directory")
	ErrSkillFileMissing = errors.New("file not found")
)

func readFileUnder(skillDir, target string) ([]byte, error) {
	skillDir = filepath.Clean(skillDir)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(skillDir, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, ErrSkillFileOutside
	}
	data, err := readSkillFileOS(target)
	if err != nil {
		return nil, ErrSkillFileMissing
	}
	return data, nil
}
