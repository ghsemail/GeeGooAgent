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

const playbookSignalProbe = "strategy-signal-probe"

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
		return playbookSignalProbe, true
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
	case playbookSignalProbe:
		return r.runSignalProbe(ctx, in)
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

	signals, err := r.resolveSignals(ctx, toolCtx, plan, recordTool)
	if err != nil {
		return runtime.TurnResult{}, false
	}

	strategyKind := strings.TrimSpace(plan.SignalKind)
	if strategyKind == "" {
		strategyKind = "combination"
	}
	tradeConfig := defaultSmartTradeTradeConfig()
	baseOrderSize := 100
	runArgs := map[string]any{
		"code":            code,
		"frequency":       signals.Frequency,
		"buy_signal":      signals.Buy,
		"sell_signal":     signals.Sell,
		"strategy_label":  signals.StrategyLabel,
		"strategy_kind":   strategyKind,
		"stock_name":      name,
		"market":          market,
		"fund":            plan.Fund,
		"months_back":     plan.MonthsBack,
		"base_order_size": baseOrderSize,
		"period":          plan.Period,
		"trade_config":    tradeConfig,
	}
	if signals.SignalID != "" {
		runArgs["strategy_ids"] = []any{signals.SignalID}
		runArgs["signal_id"] = signals.SignalID
	}
	runRes := r.runTool(ctx, toolCtx, "run_strategy_backtest", runArgs, recordTool)
	if runRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("回测执行失败：%s", runRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: runRes.Summary, StepRecords: records}, true
	}

	logDetail := map[string]any(nil)
	if logID := strings.TrimSpace(fmt.Sprint(runRes.Data["log_id"])); logID != "" {
		logRes := r.runTool(ctx, toolCtx, "get_strategy_backtest_log", map[string]any{"log_id": logID}, recordTool)
		if logRes.Status == tools.StatusOK {
			logDetail = logRes.Data
		}
	}

	reply := r.formatBacktestReply(ctx, in, step, backtestReplyInput{
		Code:          code,
		Name:          name,
		Market:        market,
		StrategyLabel: signals.StrategyLabel,
		StrategyKind:  strategyKind,
		Frequency:     signals.Frequency,
		Period:        plan.Period,
		MonthsBack:    plan.MonthsBack,
		Fund:          plan.Fund,
		BaseOrderSize: baseOrderSize,
		BuySignal:     signals.Buy,
		SellSignal:    signals.Sell,
		SignalID:      signals.SignalID,
		TradeConfig:   tradeConfig,
		RunSummary:    runRes.Data,
		LogDetail:     logDetail,
	})
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
