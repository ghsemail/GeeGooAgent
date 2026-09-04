package playbookexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ProbeRunPlan is the structured plan for signal probe SOP.
type ProbeRunPlan struct {
	StockQuery  string
	SignalQuery string
	SignalKind  string
	MonthsBack  int
	Frequency   string
}

func (r *Router) runProbe(ctx context.Context, in Input) runtime.TurnResult {
	records := []runtime.StepRecord{}
	step := in.StepBase
	if step <= 0 {
		step = 1
	}
	emit := in.OnProgress
	recordTool := func(name, status, summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: name, ToolStatus: status, Summary: summary,
		})
	}

	plan := heuristicProbePlan(in.UserText)
	enrichProbeFromSession(&plan, in.Session)
	if strings.TrimSpace(plan.StockQuery) == "" {
		msg := "请说明要测哪只标的的信号"
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: msg, StepRecords: records}
	}

	toolCtx := in.ToolCtx
	toolCtx.FullCatalogPayload = true
	toolCtx.Step = step
	if emit != nil {
		toolCtx.Progress = func(event string, data map[string]any) {
			emit(event, data)
		}
	}

	runTool := func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		return r.runTool(ctx, tc, req.Name, req.Arguments, recordTool)
	}
	code, name, _, err := slots.ResolveStock(ctx, toolCtx, runTool, plan.StockQuery)
	if err != nil {
		msg := err.Error()
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: msg, StepRecords: records}
	}

	btPlan := BacktestRunPlan{
		StockQuery:  plan.StockQuery,
		SignalQuery: plan.SignalQuery,
		SignalKind:  plan.SignalKind,
	}
	buy, sell, frequency, strategyLabel, err := r.resolveSignals(ctx, toolCtx, btPlan, recordTool)
	if err != nil {
		msg := err.Error()
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: msg, StepRecords: records}
	}
	if strings.TrimSpace(plan.Frequency) != "" {
		frequency = plan.Frequency
	}

	probeRes := r.runTool(ctx, toolCtx, "probe_bot_signal_series", map[string]any{
		"code":        code,
		"frequency":   frequency,
		"buy_signal":  buy,
		"sell_signal": sell,
		"months_back": plan.MonthsBack,
	}, recordTool)
	if probeRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("信号探测失败：%s", probeRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: probeRes.Summary, StepRecords: records}
	}

	reply := formatProbeReply(name, code, strategyLabel, frequency, probeRes)
	in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
	records = append(records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(reply, 300),
	})
	return runtime.TurnResult{AssistantText: reply, StepRecords: records}
}

func heuristicProbePlan(message string) ProbeRunPlan {
	plan := ProbeRunPlan{MonthsBack: 3, Frequency: ""}
	msg := strings.TrimSpace(message)
	plan.StockQuery = slots.ExtractStockQuery(msg)
	upper := strings.ToUpper(msg)
	switch {
	case strings.Contains(upper, "SAR") && strings.Contains(upper, "MACD"):
		plan.SignalKind, plan.SignalQuery = "combination", "SAR MACD"
	case strings.Contains(upper, "RSI"):
		plan.SignalKind, plan.SignalQuery = "indicator", "RSI"
	case strings.Contains(upper, "SAR"):
		plan.SignalKind, plan.SignalQuery = "indicator", "SAR"
	case strings.Contains(msg, "共振"):
		plan.SignalKind, plan.SignalQuery = "combination", "共振"
		plan.Frequency = "15m"
	case strings.Contains(msg, "4H") || strings.Contains(msg, "4h"):
		plan.Frequency = "60m"
	}
	return plan
}

func enrichProbeFromSession(plan *ProbeRunPlan, session *runtime.Session) {
	if plan == nil || session == nil {
		return
	}
	if strings.TrimSpace(plan.SignalQuery) == "" {
		if label := lastStrategyLabel(session); label != "" {
			plan.SignalQuery = label
			plan.SignalKind = "combination"
		}
	}
	if strings.TrimSpace(plan.StockQuery) == "" {
		if q := lastConfirmedStock(session); q != "" {
			plan.StockQuery = q
		}
	}
}

func formatProbeReply(name, code, strategyLabel, frequency string, res tools.Result) string {
	buyHits := fmt.Sprint(res.Data["buy_hits"])
	sellHits := fmt.Sprint(res.Data["sell_hits"])
	summary := strings.TrimSpace(res.Summary)
	if summary == "" {
		summary = fmt.Sprintf("买次 %s · 卖次 %s", buyHits, sellHits)
	}
	return fmt.Sprintf("## %s %s · 信号探测（%s）\n\n- **策略**：%s\n- **结果**：%s\n\n> 由 playbook 确定性执行（probe_bot_signal_series）",
		name, code, frequency, strategyLabel, summary)
}
