package workflow

import (
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Step is one workflow step.
type Step struct {
	Name      string
	Tool      string
	Arguments map[string]any
	ArgFunc   func(*memory.PreMarketWorking) map[string]any
}

// Args resolves step arguments.
func (s Step) Args(working *memory.PreMarketWorking) map[string]any {
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
	SessionID string
	Status    string
	Working   *memory.PreMarketWorking
	LastError string
}

// OK returns true when completed successfully.
func (r RunResult) OK() bool { return r.Status == "completed" }

// Runner executes deterministic workflow steps.
type Runner struct {
	executor *runtime.Executor
	working  *memory.WorkingStore
	checkpts CheckpointSaver
}

// CheckpointSaver persists checkpoints.
type CheckpointSaver interface {
	Save(sessionID, skill, status, lastTool string, step int, working *memory.PreMarketWorking) error
}

// NewRunner creates a workflow runner.
func NewRunner(executor *runtime.Executor, working *memory.WorkingStore, checkpts CheckpointSaver) *Runner {
	return &Runner{executor: executor, working: working, checkpts: checkpts}
}

// Run executes phase A then optional per-stock phase B.
func (r *Runner) Run(
	sessionID, skill string,
	phaseA, perStock []Step,
	ctx tools.Context,
	working *memory.PreMarketWorking,
) RunResult {
	return r.RunFrom(sessionID, skill, phaseA, perStock, ctx, working, 0)
}

// RunFrom resumes a workflow by skipping flattened steps up to completedStep.
func (r *Runner) RunFrom(
	sessionID, skill string,
	phaseA, perStock []Step,
	ctx tools.Context,
	working *memory.PreMarketWorking,
	completedStep int,
) RunResult {
	flatLen := len(phaseA)
	finalStep := completedStep
	for index, step := range phaseA {
		stepIndex := index + 1
		if stepIndex <= completedStep {
			continue
		}
		finalStep = stepIndex
		ctx.Step = stepIndex
		var errResult *RunResult
		working, errResult = r.processStep(sessionID, skill, step, ctx, working, stepIndex)
		if errResult != nil {
			return *errResult
		}
		if step.Tool == "check_trading_day" && working.IsTradingDay != nil && !*working.IsTradingDay {
			if err := r.checkpts.Save(sessionID, skill, "completed", step.Tool, stepIndex, working); err != nil {
				return RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
			}
			return RunResult{SessionID: sessionID, Status: "completed", Working: working}
		}
	}

	if perStock != nil && (working.IsTradingDay == nil || *working.IsTradingDay) {
		working.Phase = "phase_b"
		_ = r.working.Save(working)
		stepCounter := flatLen
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
				if stepCounter <= completedStep {
					continue
				}
				finalStep = stepCounter
				ctx.Step = stepCounter
				named := Step{Name: code + "/" + step.Name, Tool: step.Tool, ArgFunc: step.ArgFunc, Arguments: step.Arguments}
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
				if step.Tool == "list_today_reports" {
					// check already_reported via working apply
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

	if err := r.checkpts.Save(sessionID, skill, "completed", "workflow_complete", finalStep, working); err != nil {
		return RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: err.Error()}
	}
	return RunResult{SessionID: sessionID, Status: "completed", Working: working}
}

func (r *Runner) processStep(
	sessionID, skill string,
	step Step,
	ctx tools.Context,
	working *memory.PreMarketWorking,
	stepIndex int,
) (*memory.PreMarketWorking, *RunResult) {
	result := r.executor.Execute(tools.CallRequest{Name: step.Tool, Arguments: step.Args(working)}, ctx)
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
		return working, &RunResult{SessionID: sessionID, Status: "failed", Working: working, LastError: result.Summary}
	}
	return working, nil
}

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
