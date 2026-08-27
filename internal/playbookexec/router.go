// Package playbookexec runs matched playbooks through deterministic tool pipelines.
// LLM is used only for structured plan extraction and final reply formatting;
// tool selection follows fixed steps from the playbook SOP.
package playbookexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const playbookBacktestRun = "strategy-backtest-run"

// ToolRunner executes one tool call (typically agent.ToolExec.Execute).
type ToolRunner func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result

// Router dispatches deterministic playbook execution.
type Router struct {
	Gateway *llm.Gateway
	RunTool ToolRunner
}

// Input is one chat turn routed to a playbook executor.
type Input struct {
	Session       *runtime.Session
	UserText      string
	MatchedSkills []string
	ToolCtx       tools.Context
	StepBase      int
	OnProgress    runtime.ProgressFunc
}

// Route reports whether a playbook executor should handle the turn.
func Route(matchedSkills []string, userText string, session *runtime.Session) (playbook string, ok bool) {
	if procedural.BacktestDCABypass(userText) {
		return "", false
	}
	if procedural.BacktestRunIntent(userText) {
		return playbookBacktestRun, true
	}
	if procedural.BacktestContinueIntent(userText) && sessionHasBacktestContext(session) {
		return playbookBacktestRun, true
	}
	if procedural.SignalProbeIntent(userText) {
		return "", false
	}
	for _, name := range matchedSkills {
		if name == playbookBacktestRun || name == "strategy-backtest" {
			return playbookBacktestRun, true
		}
	}
	return "", false
}

// TryRun executes a deterministic playbook when Route matches.
func (r *Router) TryRun(ctx context.Context, in Input) (runtime.TurnResult, bool) {
	if r == nil || r.RunTool == nil {
		return runtime.TurnResult{}, false
	}
	playbook, ok := Route(in.MatchedSkills, in.UserText, in.Session)
	if !ok {
		return runtime.TurnResult{}, false
	}
	switch playbook {
	case playbookBacktestRun:
		return r.runBacktest(ctx, in)
	default:
		return runtime.TurnResult{}, false
	}
}

func (r *Router) runBacktest(ctx context.Context, in Input) (runtime.TurnResult, bool) {
	records := []runtime.StepRecord{}
	step := in.StepBase
	if step <= 0 {
		step = 1
	}
	emit := in.OnProgress
	recordPlan := func(summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "plan",
			Summary: strings.TrimSpace(summary),
		})
		if emit != nil {
			emit("playbook_exec", map[string]any{
				"playbook": playbookBacktestRun, "phase": "plan", "summary": summary,
			})
		}
	}
	recordTool := func(name, status, summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: name, ToolStatus: status, Summary: summary,
		})
	}

	plan, planNote, err := r.buildBacktestPlan(ctx, in, step)
	recordPlan(planNote)
	if err != nil {
		return runtime.TurnResult{}, false
	}

	toolCtx := in.ToolCtx
	toolCtx.FullCatalogPayload = true
	toolCtx.Step = step
	if emit != nil {
		toolCtx.Progress = func(event string, data map[string]any) {
			emit(event, data)
		}
	}

	code, name, market, err := r.resolveStock(ctx, toolCtx, plan, recordTool)
	if err != nil {
		return runtime.TurnResult{}, false
	}

	buy, sell, frequency, strategyLabel, err := r.resolveSignals(ctx, toolCtx, plan, recordTool)
	if err != nil {
		return runtime.TurnResult{}, false
	}

	strategyKind := strings.TrimSpace(plan.SignalKind)
	if strategyKind == "" {
		strategyKind = "combination"
	}
	runArgs := map[string]any{
		"code":            code,
		"frequency":       frequency,
		"buy_signal":      buy,
		"sell_signal":     sell,
		"strategy_label":  strategyLabel,
		"strategy_kind":   strategyKind,
		"stock_name":      name,
		"market":          market,
		"fund":            plan.Fund,
		"months_back":     plan.MonthsBack,
		"base_order_size": 100,
		"period":          plan.Period,
	}
	runRes := r.runTool(ctx, toolCtx, "run_strategy_backtest", runArgs, recordTool)
	if runRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("回测执行失败：%s", runRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: runRes.Summary, StepRecords: records}, true
	}

	reply := formatBacktestReply(code, name, strategyLabel, runRes.Data)
	in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
	records = append(records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(reply, 300),
	})
	return runtime.TurnResult{AssistantText: reply, StepRecords: records}, true
}

func (r *Router) runTool(
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
			"name": name, "status": string(res.Status), "summary": res.Summary, "arguments": args,
		})
	}
	recordTool(name, string(res.Status), res.Summary)
	return res
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}
