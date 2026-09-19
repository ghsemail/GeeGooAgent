package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ToolRunner executes one tool call.
type ToolRunner func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result

// Runner executes serial task flows inside chat turns.
type Runner struct {
	RunTool         ToolRunner
	OnProgress      runtime.ProgressFunc
	ComposeLLM      llm.Provider // optional; strategy_dev cognition synthesis
	retryFailedOnly bool
}

// RunTurn attempts to handle the user message via workflow. handled is true when ReAct should be skipped.
func (r *Runner) RunTurn(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
	stepBase int,
) (runtime.TurnResult, bool) {
	if r == nil || r.RunTool == nil || session == nil {
		return runtime.TurnResult{}, false
	}
	flow := LoadFromSession(session)
	if IsCancelIntent(userText) && flow != nil && flow.Active() {
		return r.cancelFlow(session, flow, stepBase), true
	}
	r.retryFailedOnly = strings.Contains(strings.ToLower(userText), "失败") ||
		strings.Contains(strings.ToLower(userText), "retry")
	if flow != nil && flow.Status == StatusInterrupted {
		flow.Status = StatusRunning
	}
	if ShouldStartSignalDiagnoseFlow(userText, flow) {
		flow = newSignalDiagnoseFlow(userText)
		ApplyPendingSignalDiagnoseOpts(session, flow)
		SaveToSession(session, flow)
		r.emitCard(flow)
		r.emit("workflow_started", map[string]any{
			"run_id": flow.RunID, "skill": flow.Template,
		})
	} else if !ShouldHandleFlowTurn(userText, flow) {
		switch {
		case ShouldStartGenerateStrategyCognitionFlow(userText, flow):
			flow = newGenerateStrategyCognitionFlow(userText)
		case ShouldStartStrategyDevFlow(userText, flow):
			flow = newStrategyDevFlow(userText)
		case ShouldStartMultiStrategyFlow(userText, session, flow):
			flow = newMultiStrategyFlow(userText, session)
		default:
			return runtime.TurnResult{}, false
		}
		SaveToSession(session, flow)
		r.emitCard(flow)
		r.emit("workflow_started", map[string]any{
			"run_id": flow.RunID, "skill": flow.Template,
		})
	} else if flow == nil {
		return runtime.TurnResult{}, false
	}
	if flow.Status == StatusPausedFailed && IsResumeIntent(userText) {
		flow.Status = StatusRunning
	}

	switch canonicalSkill(flow.Template) {
	case SkillGenerateStrategyArchive:
		return r.runTemplateTurn(ctx, session, flow, toolCtx, stepBase, r.advanceGenerateStrategyCognition, renderGenerateCognitionPartial, renderGenerateCognitionReport)
	case SkillStrategyDev:
		return r.runTemplateTurn(ctx, session, flow, toolCtx, stepBase, r.advanceStrategyDev, renderStrategyDevPartial, renderStrategyDevReport)
	case SkillSignalDiagnose:
		return r.runTemplateTurn(ctx, session, flow, toolCtx, stepBase, r.advanceSignalDiagnose, renderSignalDiagnosePartial, renderSignalDiagnoseReport)
	case SkillMultiStrategyCompare:
		return r.runTemplateTurn(ctx, session, flow, toolCtx, stepBase, r.advanceMultiStrategy, renderPartialReport, renderFinalReport)
	default:
		return runtime.TurnResult{}, false
	}
}

type advanceFunc func(ctx context.Context, session *runtime.Session, flow *Flow, toolCtx tools.Context, recordTool func(name, status, summary string)) error
type reportPartialFunc func(*Flow) string
type reportFinalFunc func(*Flow) string

func (r *Runner) runTemplateTurn(
	ctx context.Context,
	session *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	stepBase int,
	advance advanceFunc,
	partial reportPartialFunc,
	final reportFinalFunc,
) (runtime.TurnResult, bool) {
	records := []runtime.StepRecord{}
	step := stepBase
	if step <= 0 {
		step = 1
	}
	recordTool := func(name, status, summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: name, ToolStatus: status, Summary: summary,
		})
	}
	toolCtx.FullCatalogPayload = true
	toolCtx.Step = step
	if r.OnProgress != nil {
		toolCtx.Progress = func(event string, data map[string]any) {
			r.OnProgress(event, data)
		}
	}
	r.emitCard(flow)
	for flow.Phase != PhaseSummarize && flow.Phase != PhaseDone {
		if err := ctx.Err(); err != nil {
			flow.Status = StatusInterrupted
			SaveToSession(session, flow)
			return runtime.TurnResult{Failed: true, Error: err.Error(), StepRecords: records}, true
		}
		if err := advance(ctx, session, flow, toolCtx, recordTool); err != nil {
			msg := fmt.Sprintf("Workflow 已暂停：%v", err)
			flow.Status = StatusPausedFailed
			flow.PartialReport = partial(flow)
			flow.touch()
			SaveToSession(session, flow)
			r.emitCard(flow)
			session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg + "\n\n" + flow.PartialReport})
			records = append(records, runtime.StepRecord{
				Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(msg, 300),
			})
			return runtime.TurnResult{
				AssistantText: msg + "\n\n" + flow.PartialReport,
				Failed:        true,
				Error:         err.Error(),
				StepRecords:   records,
			}, true
		}
	}
	if flow.Phase == PhaseSummarize {
		reply := final(flow)
		flow.Phase = PhaseDone
		flow.Status = StatusCompleted
		flow.PartialReport = reply
		flow.touch()
		SaveToSession(session, nil)
		session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(reply, 300),
		})
		r.emitCard(flow)
		r.emit("workflow_completed", map[string]any{"run_id": flow.RunID})
		return runtime.TurnResult{AssistantText: reply, StepRecords: records}, true
	}
	flow.PartialReport = partial(flow)
	flow.touch()
	SaveToSession(session, flow)
	r.emitCard(flow)
	reply := flow.PartialReport
	if strings.TrimSpace(reply) == "" {
		reply = "Workflow 进行中，请发送「继续」以执行下一步。"
	}
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
	return runtime.TurnResult{AssistantText: reply, StepRecords: records}, true
}

func (r *Runner) cancelFlow(session *runtime.Session, flow *Flow, step int) runtime.TurnResult {
	flow.Status = StatusCancelled
	flow.Phase = PhaseDone
	flow.touch()
	SaveToSession(session, nil)
	msg := "已取消当前 Workflow。"
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
	return runtime.TurnResult{
		AssistantText: msg,
		StepRecords: []runtime.StepRecord{{
			Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: msg,
		}},
	}
}

func (r *Runner) emit(event string, data map[string]any) {
	if r.OnProgress != nil {
		r.OnProgress(event, data)
	}
}

func (r *Runner) emitCard(flow *Flow) {
	if flow == nil {
		return
	}
	r.emit("workflow_card", CardPayload(flow))
}

func (r *Runner) runTool(
	ctx context.Context,
	toolCtx tools.Context,
	name string,
	args map[string]any,
	recordTool func(name, status, summary string),
) tools.Result {
	if toolCtx.Progress != nil {
		toolCtx.Progress("tool_start", map[string]any{"name": name, "arguments": args})
	}
	res := r.RunTool(ctx, tools.CallRequest{Name: name, Arguments: args}, toolCtx)
	if toolCtx.Progress != nil {
		toolCtx.Progress("tool_done", map[string]any{
			"name": name, "status": string(res.Status), "summary": res.Summary,
		})
	}
	if recordTool != nil {
		recordTool(name, string(res.Status), res.Summary)
	}
	return res
}

func (r *Runner) runToolCall(recordTool func(name, status, summary string)) ToolRunner {
	return func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
		return r.runTool(ctx, toolCtx, req.Name, req.Arguments, recordTool)
	}
}

// MarkInterrupted should be called when a running flow is loaded after process restart.
func MarkInterrupted(flow *Flow) {
	if flow == nil {
		return
	}
	if flow.Status == StatusRunning {
		flow.Status = StatusInterrupted
		flow.touch()
	}
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}
