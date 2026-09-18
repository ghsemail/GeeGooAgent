package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ScheduledResult is the outcome of a cron/batch chat workflow run.
type ScheduledResult struct {
	SessionID string
	Status    string // completed | failed
	LastError string
	Verdict   string // pass | fail
	Report    string
}

// RunScheduled executes a chat workflow skill to completion (Scheduler cron / CLI).
func RunScheduled(
	ctx context.Context,
	skill string,
	prompt string,
	session *runtime.Session,
	toolCtx tools.Context,
	runTool ToolRunner,
) (ScheduledResult, error) {
	if runTool == nil {
		return ScheduledResult{}, fmt.Errorf("tool runner required")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ScheduledResult{}, fmt.Errorf("workflow %s requires prompt (stock + optional strategies)", skill)
	}
	if session == nil {
		session = &runtime.Session{ID: newRunID()}
	}
	r := &Runner{RunTool: runTool}
	switch skill {
	case SkillMultiStrategyCompare:
		return r.runMultiStrategyBatch(ctx, session, prompt, toolCtx)
	default:
		return ScheduledResult{}, fmt.Errorf("workflow %s is not schedulable yet", skill)
	}
}

func (r *Runner) runMultiStrategyBatch(
	ctx context.Context,
	session *runtime.Session,
	prompt string,
	toolCtx tools.Context,
) (ScheduledResult, error) {
	flow := newMultiStrategyFlow(prompt, session)
	if flow == nil || flow.Template == "" {
		return ScheduledResult{Status: "failed", LastError: "invalid flow", Verdict: "fail"},
			fmt.Errorf("failed to start workflow")
	}
	recordTool := func(string, string, string) {}
	toolCtx.FullCatalogPayload = true
	for flow.Phase != PhaseSummarize && flow.Phase != PhaseDone {
		if err := ctx.Err(); err != nil {
			return ScheduledResult{Status: "failed", LastError: err.Error(), Verdict: "fail"}, err
		}
		if err := r.advanceMultiStrategy(ctx, session, flow, toolCtx, recordTool); err != nil {
			return ScheduledResult{
				SessionID: session.ID,
				Status:    "failed",
				LastError: err.Error(),
				Verdict:   "fail",
			}, err
		}
	}
	report := renderFinalReport(flow)
	return ScheduledResult{
		SessionID: session.ID,
		Status:    "completed",
		Verdict:   "pass",
		Report:    report,
	}, nil
}
