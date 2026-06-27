package runtime

import (
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Executor dispatches tool calls to the registry.
type Executor struct {
	registry *tools.Registry
}

// NewExecutor creates a tool executor.
func NewExecutor(registry *tools.Registry) *Executor {
	return &Executor{registry: registry}
}

// Execute runs one tool call.
func (e *Executor) Execute(req tools.CallRequest, ctx tools.Context) tools.Result {
	return e.registry.Execute(req, ctx)
}
