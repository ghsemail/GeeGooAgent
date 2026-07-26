package procedural

import (
	"sort"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// SkillsPolicy filters procedural skills for chat injection and dashboard.
type SkillsPolicy struct {
	disabled    map[string]struct{}
	extensions  map[string]bool // name -> enabled; missing => true
}

// PolicyFromConfig builds runtime policy from app config.
func PolicyFromConfig(cfg *config.SkillsConfig) SkillsPolicy {
	p := SkillsPolicy{
		disabled:   map[string]struct{}{},
		extensions: map[string]bool{},
	}
	if cfg == nil {
		return p
	}
	for _, name := range cfg.Disabled {
		name = normalizeSkillName(name)
		if name != "" {
			p.disabled[name] = struct{}{}
		}
	}
	for name, ext := range cfg.Extensions {
		name = normalizeSkillName(name)
		if name == "" {
			continue
		}
		enabled := true
		if ext.Enabled != nil {
			enabled = *ext.Enabled
		}
		p.extensions[name] = enabled
	}
	return p
}

// DisabledNames returns skill names explicitly disabled in config.
func (p SkillsPolicy) DisabledNames() []string {
	out := make([]string, 0, len(p.disabled))
	for name := range p.disabled {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Enabled reports whether a skill should be active (inject + extension scripts).
func (p SkillsPolicy) Enabled(name string, kind Kind) bool {
	name = normalizeSkillName(name)
	if name == "" {
		return false
	}
	if _, off := p.disabled[name]; off {
		return false
	}
	if kind == KindExtension {
		if v, ok := p.extensions[name]; ok {
			return v
		}
	}
	return true
}

// ExtensionEnabled is shorthand for extension script runners.
func (p SkillsPolicy) ExtensionEnabled(name string) bool {
	return p.Enabled(name, KindExtension)
}

func normalizeSkillName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
