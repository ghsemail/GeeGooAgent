// Package agent is the platform-agnostic core of GeeGooAgent.
//
// It owns the ReAct loop, LLM gateway, tool executor, and tool registry,
// exposing a single Run entry point used by CLI chat, the HTTP runtime,
// and (in later phases) the workflow runner and scheduler. Platform
// differences live in the entry points (cmd/geegoo, agent-runtime), not
// inside the agent.
package agent

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Agent bundles the loop and its collaborators. It is intentionally thin:
// today it delegates to runtime.ReActLoop. As later phases add prompt
// building, compression, and trajectory capture, those responsibilities
// migrate here so every entry point shares them.
type Agent struct {
	Loop     *runtime.ReActLoop
	Gateway  *llm.Gateway
	Executor *runtime.Executor
	Registry *tools.Registry
}

// New constructs an Agent from the supplied collaborators.
func New(gateway *llm.Gateway, executor *runtime.Executor, registry *tools.Registry) *Agent {
	return &Agent{
		Loop: runtime.NewReActLoop(gateway, executor), Gateway: gateway,
		Executor: executor, Registry: registry,
	}
}

// Run executes one user turn through the ReAct loop. ctx governs
// cancellation of LLM and tool calls.
func (a *Agent) Run(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
) runtime.TurnResult {
	return a.Loop.RunTurn(ctx, session, userText, toolCtx, schemas)
}

// SetProgress wires live step output (used by chat UI).
func (a *Agent) SetProgress(fn runtime.ProgressFunc) {
	if a.Loop != nil {
		a.Loop.SetProgress(fn)
	}
}
