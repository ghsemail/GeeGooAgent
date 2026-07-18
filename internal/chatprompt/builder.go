package chatprompt

import "strings"

// SystemBuilder assembles the stable system prompt from layered sections
// (Hermes prompt_builder parity: soul + tools + memory + endpoints).
type SystemBuilder struct {
	Sections []string
}

// DefaultBuilder returns the production system prompt builder.
func DefaultBuilder() SystemBuilder {
	return SystemBuilder{Sections: []string{
		Soul(),
		ToolRouting(),
		MemoryRules(),
		ServiceEndpoints(),
	}}
}

// Build joins non-empty sections with blank lines.
func (b SystemBuilder) Build() string {
	var parts []string
	for _, s := range b.Sections {
		if t := strings.TrimSpace(s); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n\n")
}
