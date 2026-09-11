package tools

import (
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// SchemaOptions controls LLM-visible tool schema export (policy v2 + defer load).
type SchemaOptions struct {
	Names            []string
	ExcludeForbidden bool
	ExcludeDeferLoad bool
	IncludeForbidden bool
	CoreOnly         bool
	ActiveToolsets   []string
	AlwaysInclude    []string
}

// SchemasWithOptions returns tool schemas after policy and defer-load filtering.
func (r *Registry) SchemasWithOptions(opts SchemaOptions) []llm.ToolSchema {
	names := r.resolveSchemaNames(opts)
	out := make([]llm.ToolSchema, 0, len(names))
	for _, name := range names {
		t, ok := r.tools[name]
		if !ok {
			continue
		}
		params := t.Parameters
		if params == nil {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		out = append(out, llm.ToolSchema{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		})
	}
	return out
}

func (r *Registry) resolveSchemaNames(opts SchemaOptions) []string {
	if r == nil {
		return nil
	}
	base := opts.Names
	if len(base) == 0 {
		base = r.ListNames()
	}
	set := make(map[string]struct{}, len(base))
	for _, name := range base {
		set[name] = struct{}{}
	}
	for _, tsID := range opts.ActiveToolsets {
		ts, ok := ToolsetByID(tsID)
		if !ok {
			continue
		}
		for _, name := range ts.Names() {
			set[name] = struct{}{}
		}
	}
	for _, name := range opts.AlwaysInclude {
		set[name] = struct{}{}
	}
	if opts.CoreOnly {
		core := CoreChatToolSet()
		filtered := make(map[string]struct{}, len(core))
		for name := range set {
			if _, ok := core[name]; ok {
				filtered[name] = struct{}{}
			}
		}
		for name := range core {
			filtered[name] = struct{}{}
		}
		set = filtered
	}
	activated := activatedToolsetNames(opts.ActiveToolsets)
	out := make([]string, 0, len(set))
	for name := range set {
		t, ok := r.tools[name]
		if !ok {
			continue
		}
		if opts.ExcludeForbidden && !opts.IncludeForbidden && t.resolved.Policy == PolicyForbidden {
			continue
		}
		if opts.ExcludeDeferLoad && t.resolved.DeferLoad {
			if _, ok := activated[name]; !ok && !isDiscoveryMetaTool(name) {
				continue
			}
		}
		out = append(out, name)
	}
	return r.sortedNames(out)
}

func activatedToolsetNames(ids []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, tsID := range ids {
		ts, ok := ToolsetByID(tsID)
		if !ok {
			continue
		}
		for _, name := range ts.Names() {
			out[name] = struct{}{}
		}
	}
	return out
}

func isDiscoveryMetaTool(name string) bool {
	return name == "discover_tools" || name == "activate_toolset"
}
