package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Step is one workflow step.
type Step struct {
	Name           string
	Tool           string
	Arguments      map[string]any
	ArgFunc        func(*memory.PreMarketWorking) map[string]any
	ContextArgFunc func(context.Context, *memory.PreMarketWorking) map[string]any
}

// Args resolves step arguments.
func (s Step) Args(ctx context.Context, working *memory.PreMarketWorking) map[string]any {
	if s.ContextArgFunc != nil {
		return s.ContextArgFunc(ctx, working)
	}
	if s.ArgFunc != nil {
		return s.ArgFunc(working)
	}
	if s.Arguments == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, v := range s.Arguments {
		out[k] = v
	}
	return out
}

// RunResult is workflow outcome.
type RunResult struct {
	SessionID  string
	Status     string
	Working    *memory.PreMarketWorking
	LastError  string
	Supervisor *SupervisorReport
}

// OK returns true when completed successfully.
func (r RunResult) OK() bool { return r.Status == "completed" }

// Runner executes deterministic workflow steps.
type Runner struct {
	executor    *runtime.Executor
	working     *memory.WorkingStore
	checkpts    CheckpointSaver
	synthesizer SynthesizerProvider
}

// SynthesizerProvider abstracts report.Synthesizer so the workflow package
// does not import report (avoids a cycle). Implementations return
// (reason, suggestion, summary, error). A nil/absent provider means the
// rule-based report path is used.
type SynthesizerProvider interface {
	Synthesize(ctx context.Context, ws memory.StockWorkspace, evidence []memory.EvidenceRef, mc memory.MarketContext) (reason, suggestion, summary string, err error)
}

// CheckpointSaver persists checkpoints.
type CheckpointSaver interface {
	Save(sessionID, skill, status, lastTool string, step int, working *memory.PreMarketWorking) error
}

// NewRunner creates a workflow runner.
func NewRunner(executor *runtime.Executor, working *memory.WorkingStore, checkpts CheckpointSaver) *Runner {
	return &Runner{executor: executor, working: working, checkpts: checkpts}
}

// SetSynthesizer wires an LLM report synthesizer. Optional; when nil the
// rule-based report content path is used.
func (r *Runner) SetSynthesizer(s SynthesizerProvider) { r.synthesizer = s }

// Run executes phase A then optional per-stock phase B, then runs supervisor.
func (r *Runner) Run(
	sessionID, skill string,
	phaseA, perStock []Step,
	ctx tools.Context,
	working *memory.PreMarketWorking,
) RunResult {
	return r.finishWithSupervisor(r.RunFrom(sessionID, skill, phaseA, perStock, ctx, working, 0), ctx)
}

// RunFrom resumes a workflow. completedStep is retained for backward
// compatibility but idempotency is now driven by working.CompletedStepKeys:
// a step is skipped when its named key is already present. This means bot
// list reordering between runs no longer causes step-index drift.
func (r *Runner) RunFrom(
	sessionID, skill string,
	phaseA, perStock []Step,
	ctx tools.Context,
	working *memory.PreMarketWorking,
	completedStep int,
) RunResult {
	_ = completedStep // idempotency via CompletedStepKeys now
	stepCounter := 0
	for index, step := range phaseA {
		stepCounter = index + 1
		key := stepKey(step.Name, step.Tool)
		if isStepComplete(working, key) {
			continue
		}
		ctx.Step = stepCounter
		var errResult *RunResult
		working, errResult = r.processStep(sessionID, skill, step, ctx, working, stepCounter)
		if errResult != nil {
			return *errResult
		}
		if step.Tool == "check_trading_day" && working.IsTradingDay != nil && !*working.IsTradingDay {
			if err := r.checkpts.Save(sessionID, skill, "completed", step.Tool, stepCounter, working); err != nil {
				return RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
			}
			return RunResult{SessionID: sessionID, Status: "completed", Working: working}
		}
	}

	if perStock != nil && (working.IsTradingDay == nil || *working.IsTradingDay) {
		working.Phase = "phase_b"
		_ = r.working.Save(working)
		for _, bot := range working.BotCodes {
			code := bot.Code
			ws, ok := working.Stocks[code]
			if !ok || ws.Status == "reported" || ws.Status == "skipped" {
				continue
			}
			working.CurrentStock = code
			if working.Stocks[code].Status == "pending" {
				ws.Status = "collecting"
				working.Stocks[code] = ws
			}
			_ = r.working.Save(working)

			skipStock := false
			for _, step := range perStock {
				if skipStock {
					break
				}
				stepCounter++
				named := Step{Name: code + "/" + step.Name, Tool: step.Tool, ArgFunc: step.ArgFunc, ContextArgFunc: step.ContextArgFunc, Arguments: step.Arguments}
				key := stepKey(named.Name, named.Tool)
				if isStepComplete(working, key) {
					continue
				}
				ctx.Step = stepCounter
				var errResult *RunResult
				working, errResult = r.processStep(sessionID, skill, named, ctx, working, stepCounter)
				if errResult != nil {
					if working.Stocks[code].Status != "failed" {
						ws := working.Stocks[code]
						ws.Status = "failed"
						working.Stocks[code] = ws
						_ = r.working.Save(working)
					}
					return *errResult
				}
				if step.Tool == "list_today_reports" || step.Tool == "list_today_post_market_reports" {
					if working.Stocks[code].Status == "skipped" {
						skipStock = true
					}
				}
			}
		}
		working.CurrentStock = ""
		allDone := true
		for _, ws := range working.Stocks {
			if ws.Status != "reported" && ws.Status != "skipped" {
				allDone = false
				break
			}
		}
		if allDone && len(working.Stocks) > 0 {
			working.Phase = "done"
		}
		_ = r.working.Save(working)
	}

	if err := r.checkpts.Save(sessionID, skill, "completed", "workflow_complete", stepCounter, working); err != nil {
		return RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
	}
	return RunResult{SessionID: sessionID, Status: "completed", Working: working}
}

// finishWithSupervisor runs acceptance checks on a completed (or failed) run
// and attaches the report. Verdict does not flip a failed status to completed.
func (r *Runner) finishWithSupervisor(result RunResult, ctx tools.Context) RunResult {
	if result.Working == nil {
		return result
	}
	eng := NewEngine(ctx.WorkspaceRoot, SupervisorChecksForSkill(result.Working.Skill))
	report := eng.Verify(result.Working, time.Now().Format("2006-01-02"))
	result.Supervisor = &report
	if ctx.EventBus != nil {
		ctx.EventBus.Emit("SupervisorVerified", map[string]any{
			"session_id": result.SessionID, "verdict": string(report.Verdict),
			"summary": report.Summary(),
		})
	}
	return result
}

func (r *Runner) processStep(
	sessionID, skill string,
	step Step,
	ctx tools.Context,
	working *memory.PreMarketWorking,
	stepIndex int,
) (*memory.PreMarketWorking, *RunResult) {
	goCtx := ctx.GoContext()
	if err := goCtx.Err(); err != nil {
		return working, &RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
	}
	result := r.executor.Execute(tools.CallRequest{Name: step.Tool, Arguments: step.Args(goCtx, working)}, ctx)
	var err error
	working, err = r.working.Apply(working, step.Tool, result)
	if err != nil {
		return working, &RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
	}
	if step.Tool != "write_execution_log" {
		logResult := r.executor.Execute(tools.CallRequest{
			Name: "write_execution_log",
			Arguments: map[string]any{
				"step": step.Name, "message": result.Summary, "status": string(result.Status),
			},
		}, ctx)
		working, _ = r.working.Apply(working, "write_execution_log", logResult)
	}
	if err := r.checkpts.Save(sessionID, skill, "running", step.Tool, stepIndex, working); err != nil {
		return working, &RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
	}
	if result.Status == tools.StatusError {
		kind := classifyError(step.Tool, result.Summary)
		if kind == ErrorRecoverable {
			// One retry for transient errors before giving up.
			if err := goCtx.Err(); err != nil {
				return working, &RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
			}
			result2 := r.executor.Execute(tools.CallRequest{Name: step.Tool, Arguments: step.Args(goCtx, working)}, ctx)
			working, _ = r.working.Apply(working, step.Tool, result2)
			if result2.Status != tools.StatusError {
				markStepComplete(working, stepKey(step.Name, step.Tool))
				return working, nil
			}
			return working, &RunResult{
				SessionID: sessionID, Status: "failed", Working: working,
				LastError: (&StepError{Kind: ErrorRecoverable, Tool: step.Tool, Message: result2.Summary}).Error(),
			}
		}
		return working, &RunResult{
			SessionID: sessionID, Status: "failed", Working: working,
			LastError: (&StepError{Kind: ErrorTerminal, Tool: step.Tool, Message: result.Summary}).Error(),
		}
	}
	markStepComplete(working, stepKey(step.Name, step.Tool))
	return working, nil
}

// stepKey is the idempotency key for a workflow step.
func stepKey(name, tool string) string {
	if name == "" {
		return tool
	}
	return name
}

func isStepComplete(working *memory.PreMarketWorking, key string) bool {
	for _, k := range working.CompletedStepKeys {
		if k == key {
			return true
		}
	}
	return false
}

func markStepComplete(working *memory.PreMarketWorking, key string) {
	if isStepComplete(working, key) {
		return
	}
	working.CompletedStepKeys = append(working.CompletedStepKeys, key)
}

// Test-only accessors for idempotency helpers.
func IsStepCompleteForTest(w *memory.PreMarketWorking, key string) bool {
	return isStepComplete(w, key)
}
func MarkStepCompleteForTest(w *memory.PreMarketWorking, key string) { markStepComplete(w, key) }
func StepKeyForTest(name, tool string) string                        { return stepKey(name, tool) }

// CheckpointAdapter bridges infra checkpoint manager.
type CheckpointAdapter struct {
	SaveFn func(sessionID, skill, status, lastTool string, step int, working *memory.PreMarketWorking) error
}

func (a CheckpointAdapter) Save(sessionID, skill, status, lastTool string, step int, working *memory.PreMarketWorking) error {
	if a.SaveFn == nil {
		return nil
	}
	return a.SaveFn(sessionID, skill, status, lastTool, step, working)
}

// WorkingLoaderAdapter loads working as map for read_working_state tool.
type WorkingLoaderAdapter struct {
	Store *memory.WorkingStore
}

func (a WorkingLoaderAdapter) Load(sessionID string) (map[string]any, error) {
	w, err := a.Store.Load(sessionID)
	if err != nil || w == nil {
		return nil, err
	}
	// reuse memory encode via store save/load roundtrip not ideal - inline minimal
	return map[string]any{"session_id": w.SessionID, "phase": w.Phase}, nil
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
