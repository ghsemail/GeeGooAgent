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
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Agent bundles the loop and its collaborators.
type Agent struct {
	Loop        *Loop
	Gateway     *llm.Gateway
	Executor    *runtime.Executor
	Registry    *tools.Registry
	reportSynth *ReportSynthesizer
	subAgent    *SubAgent
}

// New constructs an Agent from the supplied collaborators.
func New(gateway *llm.Gateway, executor *runtime.Executor, registry *tools.Registry) *Agent {
	return &Agent{
		Loop:     NewLoop(gateway, executor),
		Gateway:  gateway,
		Executor: executor,
		Registry: registry,
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

// SetCompressor wires context compression into the owned loop.
func (a *Agent) SetCompressor(c *prompt.Compressor) {
	if a != nil && a.Loop != nil {
		a.Loop.SetCompressor(c)
	}
	if a != nil && a.subAgent != nil {
		a.subAgent.SetCompressor(c)
	}
}

// SetGateway swaps the LLM gateway and keeps the owned loop in sync.
func (a *Agent) SetGateway(g *llm.Gateway) {
	if a == nil {
		return
	}
	a.Gateway = g
	if a.Loop != nil {
		a.Loop.SetGateway(g)
	}
	if a.reportSynth != nil {
		a.reportSynth.SetGateway(g)
	}
}

// SetProgress wires live step output (used by chat UI).
func (a *Agent) SetProgress(fn runtime.ProgressFunc) {
	if a.Loop != nil {
		a.Loop.SetProgress(fn)
	}
}

// SetMaxToolRounds sets the per-turn LLM↔tool iteration cap (config max_steps).
func (a *Agent) SetMaxToolRounds(n int) {
	if a.Loop != nil {
		a.Loop.SetMaxToolRounds(n)
	}
}

// SetToolMaxParallel caps concurrent tool calls per LLM round.
func (a *Agent) SetToolMaxParallel(n int) {
	if a.Loop != nil {
		a.Loop.SetToolMaxParallel(n)
	}
}

// SetToolTimeout bounds a single tool invocation.
func (a *Agent) SetToolTimeout(d time.Duration) {
	if a.Loop != nil {
		a.Loop.SetToolTimeout(d)
	}
}

// SetApproval wires interactive confirmation for mutating tools.
func (a *Agent) SetApproval(fn runtime.ApprovalFunc) {
	if a.Loop != nil {
		a.Loop.SetApproval(fn)
	}
	if a.subAgent != nil {
		a.subAgent.SetApproval(fn)
	}
}

// SetSubAgent wires the delegate_task runner (approval/compressor/event bus sync via Agent setters).
func (a *Agent) SetSubAgent(sub *SubAgent) {
	if a == nil {
		return
	}
	a.subAgent = sub
}

// SetEventBus wires turn-level observability on the loop.
func (a *Agent) SetEventBus(bus tools.EventEmitter) {
	if a.Loop != nil {
		a.Loop.SetEventBus(bus)
	}
	if a.subAgent != nil {
		a.subAgent.SetEventBus(bus)
	}
}

// SetReportSynthesizer wires workflow report LLM synthesis (shared gateway).
func (a *Agent) SetReportSynthesizer(s *ReportSynthesizer) {
	if a == nil {
		return
	}
	a.reportSynth = s
	if a.Gateway != nil && s != nil {
		s.SetGateway(a.Gateway)
	}
}

// ReportSynthesizer returns the evidence-only report synthesizer.
func (a *Agent) ReportSynthesizer() *ReportSynthesizer {
	if a == nil {
		return nil
	}
	return a.reportSynth
}

// ToolExec returns the shared tool dispatcher for workflow and other callers.
func (a *Agent) ToolExec() *ToolExec {
	if a == nil || a.Loop == nil {
		return nil
	}
	return a.Loop.ToolExec()
}
