package tools

import (
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
	SessionID     string
	MCPToken      string
	DryRun        bool
	Step          int
	WorkspaceRoot string
	EventBus      EventEmitter
	StateStore    *infra.StateStore
}

// Result is returned by a tool handler.
type Result struct {
	Status   Status
	Summary  string
	Data     map[string]any
	ExitCode int
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
}

// Registry maps tool names to implementations.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Schemas returns LLM tool schemas, optionally filtered.
func (r *Registry) Schemas(filter []string) []llm.ToolSchema {
	names := r.sortedNames(filter)
	out := make([]llm.ToolSchema, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
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
	if req.Arguments == nil {
		req.Arguments = map[string]any{}
	}
	if ctx.EventBus != nil {
		ctx.EventBus.Emit("ToolCalled", map[string]any{"tool": req.Name, "step": ctx.Step})
	}
	result := t.Handle(ctx, req.Arguments)
	if ctx.EventBus != nil {
		ctx.EventBus.Emit("ToolFinished", map[string]any{
			"tool": req.Name, "step": ctx.Step, "status": string(result.Status), "summary": result.Summary,
		})
	}
	return result
}

func (r *Registry) sortedNames(filter []string) []string {
	if len(filter) == 0 {
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

