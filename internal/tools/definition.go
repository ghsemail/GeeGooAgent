package tools

import "time"

// RenderFunc customizes LLM-visible tool output (Claude renderResultForAssistant).
type RenderFunc func(toolName string, result Result, maxChars int) string

// ToolLifecycle optional hooks around tool execution (stage E).
type ToolLifecycle struct {
	BeforeCall func(ctx Context, args map[string]any) error
	AfterCall  func(ctx Context, args map[string]any, result Result)
	OnError    func(ctx Context, args map[string]any, result Result)
}

// ToolDefinition is the SSOT view of a registered tool for Registry / Loop / MCP / dashboard.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
	Execute     Handler

	ReadOnly         bool
	ConcurrencySafe  bool
	MaxResultChars   int
	Timeout          time.Duration
	DeferLoad        bool
	UserFacingLabel  string
	Policy           PolicyAction
	Domain           ToolDomain
	Toolsets         []string
	WorkflowOnly     bool
	RenderForLLM     RenderFunc
	Lifecycle        *ToolLifecycle
}

// DefinitionFromTool builds a ToolDefinition from a registered Tool.
func DefinitionFromTool(t Tool) ToolDefinition {
	meta := t.resolved
	if meta.MaxResultChars == 0 {
		meta = resolveToolMeta(t)
	}
	d := ToolDefinition{
		Name:            t.Name,
		Description:     t.Description,
		Parameters:      t.Parameters,
		Execute:         t.Handle,
		ReadOnly:        meta.ReadOnly,
		ConcurrencySafe: meta.ConcurrencySafe,
		MaxResultChars:  meta.MaxResultChars,
		Timeout:         meta.Timeout,
		DeferLoad:       meta.DeferLoad,
		UserFacingLabel: meta.UserFacingLabel,
		Policy:          meta.Policy,
		Domain:          toolDomain(t.Name),
		Toolsets:        toolsetIDsFor(t.Name),
		WorkflowOnly:    IsWorkflowExclusiveTool(t.Name),
		RenderForLLM:    t.RenderForLLM,
		Lifecycle:       t.Lifecycle,
	}
	return d
}

// Definitions returns SSOT definitions for all registered tools.
func (r *Registry) Definitions() []ToolDefinition {
	if r == nil {
		return nil
	}
	names := r.ListNames()
	out := make([]ToolDefinition, 0, len(names))
	for _, name := range names {
		if t, ok := r.tools[name]; ok {
			out = append(out, DefinitionFromTool(t))
		}
	}
	return out
}
