package procedural

import (
	"path/filepath"
	"strings"
)

// Kind classifies a SKILL.md role (what it does).
type Kind string

const (
	KindPlaybook  Kind = "playbook"
	KindWorkflow  Kind = "workflow"
	KindExtension Kind = "extension"
	KindUser      Kind = "user"
	KindOther     Kind = "other"
)

// KindBundled is deprecated; use KindExtension. Kept for JSON compatibility checks.
const KindBundled = KindExtension

// Provenance classifies where a skill came from.
type Provenance string

const (
	ProvenanceCore     Provenance = "core"
	ProvenanceOptional Provenance = "optional"
	ProvenanceMarket   Provenance = "market"
	ProvenanceTenant   Provenance = "tenant"
	ProvenanceOther    Provenance = "other"
)

// ClassifyPath infers skill kind from filesystem path.
func ClassifyPath(path string) Kind {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case IsExtensionPath(p):
		return KindExtension
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

// ClassifyProvenance infers skill provenance from path.
func ClassifyProvenance(path string) Provenance {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(p, "/.geegoo/") && strings.Contains(p, "/skills/"):
		return ProvenanceTenant
	case IsExtensionPath(p):
		return ProvenanceOptional
	case strings.Contains(p, "/playbooks/"):
		return ProvenanceCore
	case strings.Contains(p, "/skills/pre_market/"), strings.Contains(p, "skills/pre_market/"),
		strings.Contains(p, "/skills/intraday/"), strings.Contains(p, "skills/intraday/"),
		strings.Contains(p, "/skills/post_market/"), strings.Contains(p, "skills/post_market/"):
		return ProvenanceCore
	default:
		return ProvenanceOther
	}
}

// KindLabel returns a short UI label.
func KindLabel(k Kind) string {
	switch k {
	case KindPlaybook:
		return "Chat Playbook"
	case KindWorkflow:
		return "L5 Workflow"
	case KindExtension:
		return "Extension Script"
	case KindUser:
		return "User Skill"
	default:
		return "Other"
	}
}

// ProvenanceLabel returns a short UI label.
func ProvenanceLabel(p Provenance) string {
	switch p {
	case ProvenanceCore:
		return "Core"
	case ProvenanceOptional:
		return "Optional"
	case ProvenanceMarket:
		return "Market"
	case ProvenanceTenant:
		return "Tenant"
	default:
		return "Other"
	}
}

// InjectInChat reports whether Match should consider this skill for chat injection.
func InjectInChat(k Kind) bool {
	return k != KindExtension
}

// SkillSummary is the API/dashboard view of one procedural skill.
type SkillSummary struct {
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Body              string     `json:"body"`
	BodyPreview       string     `json:"body_preview"`
	Path              string     `json:"path"`
	Rel               string     `json:"rel"`
	Kind              Kind       `json:"kind"`
	KindLabel         string     `json:"kind_label"`
	Provenance        Provenance `json:"provenance"`
	ProvenanceLabel   string     `json:"provenance_label"`
	InjectInChat      bool       `json:"inject_in_chat"`
	Editable          bool       `json:"editable"`
	Enabled           bool       `json:"enabled"`
}

// Summarize builds a dashboard payload for one skill.
func Summarize(sk Skill, scanDirs []string, policy SkillsPolicy) SkillSummary {
	kind := ClassifyPath(sk.Path)
	prov := ClassifyProvenance(sk.Path)
	body := strings.TrimSpace(sk.Body)
	enabled := policy.Enabled(sk.Name, kind)
	inject := InjectInChat(kind) && enabled
	return SkillSummary{
		Name:            sk.Name,
		Description:     sk.Description,
		Body:            body,
		BodyPreview:     truncateRunes(body, 480),
		Path:            sk.Path,
		Rel:             relSkillPath(sk.Path, scanDirs),
		Kind:            kind,
		KindLabel:       KindLabel(kind),
		Provenance:      prov,
		ProvenanceLabel: ProvenanceLabel(prov),
		InjectInChat:    inject,
		Editable:        kind == KindUser,
		Enabled:         enabled,
	}
}

// MemoryConfig describes procedural memory runtime settings.
type MemoryConfig struct {
	MaxSkillsPerTurn int      `json:"max_skills_per_turn"`
	MatchMinOverlap  int      `json:"match_min_overlap"`
	ScanDirs         []string `json:"scan_dirs"`
	LoadedCount      int      `json:"loaded_count"`
	InjectableCount  int      `json:"injectable_count"`
	Disabled         []string `json:"disabled,omitempty"`
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
	policy := loader.Policy()
	if names := policy.DisabledNames(); len(names) > 0 {
		cfg.Disabled = names
	}
	all := loader.List()
	cfg.LoadedCount = len(all)
	for _, sk := range all {
		s := Summarize(sk, cfg.ScanDirs, policy)
		if s.InjectInChat {
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
		"name":               s.Name,
		"description":        s.Description,
		"body":               s.Body,
		"body_preview":       s.BodyPreview,
		"path":               s.Path,
		"rel":                s.Rel,
		"kind":               string(s.Kind),
		"kind_label":         s.KindLabel,
		"provenance":         string(s.Provenance),
		"provenance_label":   s.ProvenanceLabel,
		"inject_in_chat":     s.InjectInChat,
		"editable":           s.Editable,
		"enabled":            s.Enabled,
	}
}

// MemoryConfigMap returns JSON-friendly procedural settings.
func (c MemoryConfig) Map() map[string]any {
	out := map[string]any{
		"max_skills_per_turn": c.MaxSkillsPerTurn,
		"match_min_overlap":   c.MatchMinOverlap,
		"scan_dirs":           c.ScanDirs,
		"loaded_count":        c.LoadedCount,
		"injectable_count":    c.InjectableCount,
	}
	if len(c.Disabled) > 0 {
		out["disabled"] = c.Disabled
	}
	return out
}

// ListSummaries returns all loaded skills as dashboard summaries.
func (l *Loader) ListSummaries() []SkillSummary {
	if l == nil {
		return nil
	}
	all := l.List()
	dirs := l.Dirs()
	policy := l.Policy()
	out := make([]SkillSummary, 0, len(all))
	for _, sk := range all {
		out = append(out, Summarize(sk, dirs, policy))
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
