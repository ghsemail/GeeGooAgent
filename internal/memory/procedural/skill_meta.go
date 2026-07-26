package procedural

import (
	"path/filepath"
	"strings"
)

// Kind classifies a SKILL.md for UI and injection policy.
type Kind string

const (
	KindPlaybook Kind = "playbook"
	KindWorkflow Kind = "workflow"
	KindBundled  Kind = "bundled"
	KindUser     Kind = "user"
	KindOther    Kind = "other"
)

// ClassifyPath infers skill kind from filesystem path.
func ClassifyPath(path string) Kind {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(p, "/bundled/"):
		return KindBundled
	case strings.Contains(p, "/playbooks/"):
		return KindPlaybook
	case strings.Contains(p, "/skills/pre_market/"), strings.Contains(p, "skills/pre_market/"),
		strings.Contains(p, "/skills/intraday/"), strings.Contains(p, "skills/intraday/"),
		strings.Contains(p, "/skills/post_market/"), strings.Contains(p, "skills/post_market/"):
		return KindWorkflow
	case strings.Contains(p, "/.geegoo/") && strings.Contains(p, "/skills/"):
		return KindUser
	default:
		return KindOther
	}
}

// KindLabel returns a short UI label.
func KindLabel(k Kind) string {
	switch k {
	case KindPlaybook:
		return "Chat Playbook"
	case KindWorkflow:
		return "L5 Workflow"
	case KindBundled:
		return "Bundled Script"
	case KindUser:
		return "User Skill"
	default:
		return "Other"
	}
}

// InjectInChat reports whether Match should consider this skill for chat injection.
func InjectInChat(k Kind) bool {
	return k != KindBundled
}

// SkillSummary is the API/dashboard view of one procedural skill.
type SkillSummary struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Body         string `json:"body"`
	BodyPreview  string `json:"body_preview"`
	Path         string `json:"path"`
	Rel          string `json:"rel"`
	Kind         Kind   `json:"kind"`
	KindLabel    string `json:"kind_label"`
	InjectInChat bool   `json:"inject_in_chat"`
	Editable     bool   `json:"editable"`
}

// Summarize builds a dashboard payload for one skill.
func Summarize(sk Skill, scanDirs []string) SkillSummary {
	kind := ClassifyPath(sk.Path)
	body := strings.TrimSpace(sk.Body)
	return SkillSummary{
		Name:         sk.Name,
		Description:  sk.Description,
		Body:         body,
		BodyPreview:  truncateRunes(body, 480),
		Path:         sk.Path,
		Rel:          relSkillPath(sk.Path, scanDirs),
		Kind:         kind,
		KindLabel:    KindLabel(kind),
		InjectInChat: InjectInChat(kind),
		Editable:     kind == KindUser,
	}
}

// MemoryConfig describes procedural memory runtime settings.
type MemoryConfig struct {
	MaxSkillsPerTurn int      `json:"max_skills_per_turn"`
	MatchMinOverlap  int      `json:"match_min_overlap"`
	ScanDirs         []string `json:"scan_dirs"`
	LoadedCount      int      `json:"loaded_count"`
	InjectableCount  int      `json:"injectable_count"`
}

// BuildMemoryConfig returns settings + counts for dashboard.
func BuildMemoryConfig(loader *Loader, maxSkills int) MemoryConfig {
	if maxSkills <= 0 {
		maxSkills = DefaultMaxSkillsPerTurn
	}
	cfg := MemoryConfig{
		MaxSkillsPerTurn: maxSkills,
		MatchMinOverlap:  MinMatchOverlap,
	}
	if loader == nil {
		return cfg
	}
	cfg.ScanDirs = loader.Dirs()
	all := loader.List()
	cfg.LoadedCount = len(all)
	for _, sk := range all {
		if InjectInChat(ClassifyPath(sk.Path)) {
			cfg.InjectableCount++
		}
	}
	return cfg
}

func relSkillPath(abs string, scanDirs []string) string {
	abs = filepath.Clean(abs)
	for _, dir := range scanDirs {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "" {
			continue
		}
		if rel, err := filepath.Rel(dir, abs); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	if base := filepath.Base(filepath.Dir(abs)); base != "" && base != "." {
		return base + "/SKILL.md"
	}
	return filepath.Base(abs)
}

// Map returns a JSON-friendly dashboard payload.
func (s SkillSummary) Map() map[string]any {
	return map[string]any{
		"name":           s.Name,
		"description":    s.Description,
		"body":           s.Body,
		"body_preview":   s.BodyPreview,
		"path":           s.Path,
		"rel":            s.Rel,
		"kind":           string(s.Kind),
		"kind_label":     s.KindLabel,
		"inject_in_chat": s.InjectInChat,
		"editable":       s.Editable,
	}
}

// MemoryConfigMap returns JSON-friendly procedural settings.
func (c MemoryConfig) Map() map[string]any {
	return map[string]any{
		"max_skills_per_turn": c.MaxSkillsPerTurn,
		"match_min_overlap":   c.MatchMinOverlap,
		"scan_dirs":           c.ScanDirs,
		"loaded_count":        c.LoadedCount,
		"injectable_count":    c.InjectableCount,
	}
}

// ListSummaries returns all loaded skills as dashboard summaries.
func (l *Loader) ListSummaries() []SkillSummary {
	if l == nil {
		return nil
	}
	all := l.List()
	dirs := l.Dirs()
	out := make([]SkillSummary, 0, len(all))
	for _, sk := range all {
		out = append(out, Summarize(sk, dirs))
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
