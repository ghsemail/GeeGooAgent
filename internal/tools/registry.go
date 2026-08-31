package tools

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// EventEmitter is optional L0 bus for /flow and observability.
type EventEmitter interface {
	Emit(event string, payload map[string]any)
}

// Status is tool execution outcome.
type Status string

const (
	StatusOK     Status = "ok"
	StatusError  Status = "error"
	StatusDryRun Status = "dry_run"
	StatusSkip   Status = "skipped"
)

// Context carries dependencies for tool handlers.
type Context struct {
	Ctx            context.Context
	SessionID      string
	MCPToken       string
	DryRun         bool
	Step           int
	WorkspaceRoot  string
	EventBus       EventEmitter
	StateStore     *infra.StateStore
	// Interactive marks an ad-hoc chat session (vs deterministic workflow).
	// Mutating tools require approval when Interactive and not Approved.
	Interactive bool
	// Approved indicates the user confirmed a mutating tool call.
	Approved bool
	// DelegateDepth tracks nested delegate_task calls (max 1).
	DelegateDepth int
	// Progress receives live loop events when tools spawn sub-agents.
	Progress ProgressFunc
	// ClarifyFn blocks for clarify tool user input (interactive chat).
	ClarifyFn ClarifyFunc
	// UserID scopes memory tools and recall to the tenant owner.
	UserID string
	// FullCatalogPayload keeps full buy_signal/sell_signal on catalog list tools (playbook executor).
	FullCatalogPayload bool
	// Hooks runs optional audit scripts around tool execution.
	Hooks *HookRunner
	// ActiveToolsets points to session-level activated toolsets (discover/activate).
	ActiveToolsets *[]string
}

// ProgressFunc is the chat progress callback signature.
type ProgressFunc func(event string, data map[string]any)

// GoContext returns the embedded context.Context, defaulting to background.
func (c Context) GoContext() context.Context {
	if c.Ctx != nil {
		return c.Ctx
	}
	return context.Background()
}

// Result is returned by a tool handler.
type Result struct {
	Status   Status
	Summary  string
	Data     map[string]any
	ExitCode int
	// Meta carries observability metadata: api_code, duration_ms, retried,
	// raw_envelope, etc. Not part of the LLM-facing tool output.
	Meta map[string]any
}

// CallRequest is an executor dispatch payload.
type CallRequest struct {
	Name      string
	Arguments map[string]any
}

// Handler executes one tool.
type Handler func(ctx Context, args map[string]any) Result

// Tool describes a registered tool.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handle      Handler
	Spec        ToolSpec
	RenderForLLM RenderFunc
	Lifecycle    *ToolLifecycle
	resolved     resolvedMeta
}

// Registry maps tool names to implementations.
type Registry struct {
	tools    map[string]Tool
	platform PlatformConfig
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	t.resolved = resolveToolMeta(t)
	r.tools[t.Name] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Schemas returns LLM tool schemas, optionally filtered by name list.
func (r *Registry) Schemas(filter []string) []llm.ToolSchema {
	return r.SchemasWithOptions(SchemaOptions{Names: filter})
}

// ListNames returns registered tool names sorted.
func (r *Registry) ListNames() []string {
	return r.sortedNames(nil)
}

// Execute runs a tool by name.
func (r *Registry) Execute(req CallRequest, ctx Context) Result {
	t, ok := r.tools[req.Name]
	if !ok {
		return Result{Status: StatusError, Summary: "unknown tool: " + req.Name, ExitCode: 1}
	}
	if r.PlatformConfig().PolicyV2 && t.resolved.Policy == PolicyForbidden {
		return Result{
			Status:   StatusError,
			Summary:  "tool forbidden by policy: " + req.Name,
			Data:     map[string]any{"tool": req.Name, "policy": string(PolicyForbidden)},
			ExitCode: 1,
		}
	}
	if req.Arguments == nil {
		req.Arguments = map[string]any{}
	}
	if err := ValidateArguments(t.Parameters, req.Arguments); err != nil {
		return Result{
			Status:  StatusError,
			Summary: "参数校验失败: " + err.Error(),
			Data:    map[string]any{"tool": req.Name, "validation_error": err.Error()},
			ExitCode: 1,
		}
	}
	if t.Lifecycle != nil && t.Lifecycle.BeforeCall != nil {
		if err := t.Lifecycle.BeforeCall(ctx, req.Arguments); err != nil {
			return Result{
				Status: StatusError, Summary: err.Error(), ExitCode: 1,
				Data: map[string]any{"tool": req.Name, "lifecycle": "before_call"},
			}
		}
	}
	var hookInject string
	if ctx.Hooks != nil {
		inject, err := ctx.Hooks.RunToolBefore(ctx, req.Name, req.Arguments)
		if err != nil {
			return Result{
				Status: StatusError, Summary: err.Error(), ExitCode: 1,
				Data: map[string]any{"tool": req.Name, "hook": "tool_before"},
			}
		}
		hookInject = inject
	}
	if ctx.EventBus != nil {
		ctx.EventBus.Emit("ToolCalled", map[string]any{"tool": req.Name, "step": ctx.Step})
	}
	result := t.Handle(ctx, req.Arguments)
	if hookInject != "" {
		if result.Meta == nil {
			result.Meta = map[string]any{}
		}
		result.Meta["hook_inject"] = hookInject
	}
	if ctx.Hooks != nil {
		if inject, err := ctx.Hooks.RunToolAfter(ctx, req.Name, req.Arguments, result); err != nil {
			if t.Lifecycle != nil && t.Lifecycle.OnError != nil {
				t.Lifecycle.OnError(ctx, req.Arguments, result)
			}
			return Result{
				Status: StatusError, Summary: err.Error(), ExitCode: 1,
				Data: map[string]any{"tool": req.Name, "hook": "tool_after"},
			}
		} else if inject != "" {
			if result.Meta == nil {
				result.Meta = map[string]any{}
			}
			result.Meta["hook_inject"] = inject
		}
	}
	if t.Lifecycle != nil && t.Lifecycle.AfterCall != nil {
		t.Lifecycle.AfterCall(ctx, req.Arguments, result)
	}
	if result.Status == StatusError && t.Lifecycle != nil && t.Lifecycle.OnError != nil {
		t.Lifecycle.OnError(ctx, req.Arguments, result)
	}
	if ctx.EventBus != nil {
		ctx.EventBus.Emit("ToolFinished", map[string]any{
			"tool": req.Name, "step": ctx.Step, "status": string(result.Status), "summary": result.Summary,
		})
	}
	return result
}

func (r *Registry) sortedNames(filter []string) []string {
	if filter == nil {
		filter = make([]string, 0, len(r.tools))
		for name := range r.tools {
			filter = append(filter, name)
		}
	}
	// simple sort
	for i := 0; i < len(filter); i++ {
		for j := i + 1; j < len(filter); j++ {
			if filter[j] < filter[i] {
				filter[i], filter[j] = filter[j], filter[i]
			}
		}
	}
	return filter
}

