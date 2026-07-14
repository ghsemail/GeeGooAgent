// Package skills provides manifest-driven skill registration and lookup.
//
// A Skill is a named, runnable workflow (e.g. pre_market) with a fixed set
// of phase A and per-stock phase B steps, a report template reference, and
// supervisor checks. Skills are registered in Go (the skills/*/manifest.yaml
// files remain the human-readable source of truth) so that `geegoo run <skill>`
// can dispatch generically without hard-coding each skill name.
package skills

import (
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

// Spec describes one runnable skill.
type Spec struct {
	Name        string
	Description string
	// PhaseA returns the phase A steps for the skill.
	PhaseA func() []workflow.Step
	// PerStock returns the per-stock phase B steps for the skill.
	PerStock func() []workflow.Step
	// TemplatePath is the relative path to the report template (docs/audit).
	TemplatePath string
	// ManifestPath is the relative path to the manifest yaml (docs/audit).
	ManifestPath string
}

// Registry maps skill name to Spec.
type Registry struct {
	skills map[string]Spec
	order  []string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{skills: map[string]Spec{}}
}

// Register adds a skill. Duplicate names overwrite.
func (r *Registry) Register(s Spec) {
	if r.skills == nil {
		r.skills = map[string]Spec{}
	}
	if _, exists := r.skills[s.Name]; !exists {
		r.order = append(r.order, s.Name)
	}
	r.skills[s.Name] = s
}

// Get returns a skill by name and whether it was found.
func (r *Registry) Get(name string) (Spec, bool) {
	s, ok := r.skills[name]
	return s, ok
}

// List returns all registered skills in registration order.
func (r *Registry) List() []Spec {
	out := make([]Spec, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.skills[name])
	}
	return out
}

// Default returns a registry pre-loaded with built-in skills.
func Default() *Registry {
	r := NewRegistry()
	RegisterBuiltins(r)
	return r
}
