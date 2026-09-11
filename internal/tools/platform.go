package tools

import "github.com/ghsemail/GeeGooAgent/internal/llm"

// PlatformConfig toggles M2–M4 tool platform behavior on a registry.
type PlatformConfig struct {
	PolicyV2           bool
	DeferLoadTools     bool
	ToolFragmentInject bool
}

// SetPlatformConfig stores platform feature flags on the registry.
func (r *Registry) SetPlatformConfig(cfg PlatformConfig) {
	if r == nil {
		return
	}
	r.platform = cfg
}

// PlatformConfig returns the active platform flags.
func (r *Registry) PlatformConfig() PlatformConfig {
	if r == nil {
		return PlatformConfig{}
	}
	return r.platform
}

// ChatSchemaOptions builds schema export options for interactive chat.
func (r *Registry) ChatSchemaOptions(chatNames []string, activeToolsets []string) SchemaOptions {
	opts := SchemaOptions{
		Names:          chatNames,
		ActiveToolsets: activeToolsets,
		AlwaysInclude:  []string{"discover_tools", "activate_toolset"},
	}
	cfg := r.PlatformConfig()
	if cfg.PolicyV2 {
		opts.ExcludeForbidden = true
	}
	if cfg.DeferLoadTools {
		opts.ExcludeDeferLoad = true
		opts.CoreOnly = true
	}
	return opts
}

// ChatSchemas exports LLM schemas for chat with platform filters applied.
func (r *Registry) ChatSchemas(chatNames []string, activeToolsets []string) []llm.ToolSchema {
	return r.SchemasWithOptions(r.ChatSchemaOptions(chatNames, activeToolsets))
}
