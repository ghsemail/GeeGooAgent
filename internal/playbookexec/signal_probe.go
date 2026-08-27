package playbookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const signalProbeReplySystem = `你是 GeeGoo 策略信号测试助手。根据 JSON 为普通投资者写 Markdown 回复（中文）。

结构必须如下：
## {股票名} {代码} · 信号测试

### 测了什么
- 用了什么策略（用策略中文名）
- K 线周期、回测时间范围（用人话，如「60分钟K线、近3个月」）

### 信号统计
- 买入信号几次、卖出信号几次
- 覆盖的起止日期

### 最近买卖信号
表格列：时间 | 方向 | 参考价
最多 10 行；不足则有几条写几条

### 小结
1～2 句，说明信号是否活跃、近期偏多买还是偏卖

禁止出现：log_id、signal_id、months_back、frequency、buy_signal、JSON、param、type=flag、英文 API 字段名。
禁止编造 JSON 中没有的数字与时间。
文末单独一行：仅供参考，非投资建议。`

type signalProbeReplyInput struct {
	Name          string
	Code          string
	StrategyLabel string
	Frequency     string
	MonthsBack    int
	Probe         probeSummary
}

func (r *Router) runSignalProbe(ctx context.Context, in Input) (runtime.TurnResult, bool) {
	records := []runtime.StepRecord{}
	step := in.StepBase
	if step <= 0 {
		step = 1
	}
	emit := in.OnProgress
	recordPlan := func(summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "plan", Summary: strings.TrimSpace(summary),
		})
		if emit != nil {
			emit("playbook_exec", map[string]any{
				"playbook": playbookSignalProbe, "phase": "plan", "summary": summary,
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

	code, name, _, err := r.resolveStock(ctx, toolCtx, plan, recordTool)
	if err != nil {
		return runtime.TurnResult{}, false
	}

	signals, err := r.resolveSignals(ctx, toolCtx, plan, recordTool)
	if err != nil {
		return runtime.TurnResult{}, false
	}

	probeArgs := map[string]any{
		"code":        code,
		"frequency":   signals.Frequency,
		"buy_signal":  signals.Buy,
		"sell_signal": signals.Sell,
		"months_back": plan.MonthsBack,
	}
	probeRes := r.runTool(ctx, toolCtx, "probe_bot_signal_series", probeArgs, recordTool)
	if probeRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("信号测试失败：%s", probeRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: probeRes.Summary, StepRecords: records}, true
	}

	summary := extractProbeSummary(probeRes.Data)
	if summary.MonthsBack <= 0 {
		summary.MonthsBack = plan.MonthsBack
	}
	if summary.Frequency == "" {
		summary.Frequency = signals.Frequency
	}
	if summary.Code == "" {
		summary.Code = code
	}

	reply := r.formatSignalProbeReply(ctx, in, step, signalProbeReplyInput{
		Name:          name,
		Code:          code,
		StrategyLabel: signals.StrategyLabel,
		Frequency:     signals.Frequency,
		MonthsBack:    plan.MonthsBack,
		Probe:         summary,
	})
	in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
	records = append(records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(reply, 300),
	})
	return runtime.TurnResult{AssistantText: reply, StepRecords: records}, true
}

func (r *Router) formatSignalProbeReply(ctx context.Context, in Input, step int, bundle signalProbeReplyInput) string {
	if r != nil && r.Gateway != nil {
		payload, err := json.Marshal(buildSignalProbeLLMPayload(bundle))
		if err == nil {
			messages := []llm.Message{
				{Role: llm.RoleSystem, Content: signalProbeReplySystem},
				{Role: llm.RoleUser, Content: "信号测试数据 JSON：\n" + string(payload)},
			}
			callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskChat, ToolSchemaCount: 0})
			resp, err := r.Gateway.ChatStream(callCtx, messages, nil, in.Session.ID, step, nil)
			if err == nil {
				if text := strings.TrimSpace(stripReasoningTags(resp.Content)); text != "" {
					return text
				}
			}
		}
	}
	return formatSignalProbeReplyFallback(bundle)
}

func buildSignalProbeLLMPayload(bundle signalProbeReplyInput) map[string]any {
	p := bundle.Probe
	return map[string]any{
		"stock": map[string]any{
			"name": bundle.Name,
			"code": bundle.Code,
		},
		"strategy": humanSignalRules(nil, nil, bundle.StrategyLabel),
		"settings": map[string]any{
			"kline":      humanFrequency(bundle.Frequency),
			"time_range": humanMonthsRange(bundle.MonthsBack),
		},
		"stats": map[string]any{
			"buy_signals":  p.BuyHits,
			"sell_signals": p.SellHits,
			"from_date":    p.RangeStart,
			"to_date":      p.RangeEnd,
			"bar_count":    p.BarCount,
		},
		"recent_signals": p.RecentSignals,
	}
}

func formatSignalProbeReplyFallback(bundle signalProbeReplyInput) string {
	p := bundle.Probe
	var b strings.Builder
	fmt.Fprintf(&b, "## %s %s · 信号测试\n\n", bundle.Name, bundle.Code)
	b.WriteString("### 测了什么\n\n")
	fmt.Fprintf(&b, "- **策略**：%s\n", humanSignalRules(nil, nil, bundle.StrategyLabel))
	fmt.Fprintf(&b, "- **K 线**：%s · %s\n\n", humanFrequency(bundle.Frequency), humanMonthsRange(bundle.MonthsBack))

	b.WriteString("### 信号统计\n\n")
	fmt.Fprintf(&b, "- **买入信号**：%d 次\n", p.BuyHits)
	fmt.Fprintf(&b, "- **卖出信号**：%d 次\n", p.SellHits)
	if p.RangeStart != "" && p.RangeEnd != "" {
		fmt.Fprintf(&b, "- **覆盖区间**：%s ～ %s\n\n", p.RangeStart, p.RangeEnd)
	} else {
		b.WriteString("\n")
	}

	if table := formatRecentSignalsTable(p.RecentSignals); table != "" {
		b.WriteString("### 最近买卖信号\n\n")
		b.WriteString(table)
		b.WriteString("\n\n")
	}

	b.WriteString("### 小结\n\n")
	if p.BuyHits == 0 && p.SellHits == 0 {
		b.WriteString("该区间内未出现买卖信号，可尝试延长观察时间或检查策略是否匹配该标的。\n\n")
	} else {
		fmt.Fprintf(&b, "共出现 %d 次买入、%d 次卖出信号。", p.BuyHits, p.SellHits)
		if len(p.RecentBuys) > 0 {
			fmt.Fprintf(&b, " 最近一次买入在 %s。", p.RecentBuys[0])
		}
		b.WriteString("\n\n")
	}
	b.WriteString("> 仅供参考，非投资建议。")
	return b.String()
}

func formatRecentSignalsTable(signals []map[string]any) string {
	if len(signals) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("| 时间 | 方向 | 参考价 |\n| --- | --- | --- |\n")
	limit := len(signals)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		row := signals[i]
		fmt.Fprintf(&b, "| %v | %v | %v |\n", row["time"], row["direction"], row["close"])
	}
	if len(signals) > limit {
		fmt.Fprintf(&b, "\n共 %d 条信号，仅列最近 %d 条。", len(signals), limit)
	}
	return strings.TrimSpace(b.String())
}
