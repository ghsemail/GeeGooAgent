package agent

import (
	"context"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const budgetSummaryUserPrompt = `[BUDGET_EXHAUSTED] 本回合已达到模型调用上限，无法继续调用工具。
请根据当前对话与已有工具结果，用中文给出阶段性结论：已确认的事实、未完成项、建议用户下一步如何缩小问题。不要编造数据，不要声称调用了新工具。`

func (l *Loop) finishBudgetExhausted(
	ctx context.Context,
	session *runtime.Session,
	messages []llm.Message,
	records []runtime.StepRecord,
) runtime.TurnResult {
	l.emit("budget_exhausted", map[string]any{
		"max_rounds": l.maxToolRounds, "steps": len(records),
	})
	l.emitBus("TurnBudgetExhausted", map[string]any{
		"session_id": session.ID, "max_rounds": l.maxToolRounds,
	})

	text := l.requestBudgetSummary(ctx, session, messages)
	if strings.TrimSpace(text) == "" {
		text = summarizeFromStepRecords(records)
	}
	if strings.TrimSpace(text) == "" {
		text = "已达到单轮 Tool 调用上限，请缩小问题范围后重试。"
	} else if !strings.Contains(text, "调用上限") {
		text = strings.TrimSpace(text) + "\n\n（已达本回合模型调用上限，以上为阶段性结论。）"
	}

	session.StepCounter++
	step := session.StepCounter
	l.emit("reply_start", map[string]any{"step": step, "budget_summary": true})
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: text})

	summary := text
	if len(summary) > 300 {
		summary = summary[:300]
	}
	records = append(records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: summary,
	})
	l.emitBus("TurnCompleted", map[string]any{
		"session_id": session.ID, "steps": len(records), "budget_exhausted": true,
	})
	return runtime.TurnResult{
		AssistantText: text,
		Failed:        true,
		Error:         "max_tool_rounds",
		StepRecords:   records,
	}
}

func (l *Loop) requestBudgetSummary(ctx context.Context, session *runtime.Session, messages []llm.Message) string {
	if l == nil || l.gateway == nil {
		return ""
	}
	if err := ctx.Err(); err != nil {
		return ""
	}
	out := append([]llm.Message(nil), messages...)
	out = append(out, llm.Message{Role: llm.RoleUser, Content: budgetSummaryUserPrompt})
	out = llm.SanitizeMessages(out)

	session.StepCounter++
	step := session.StepCounter
	onDelta := l.streamHandler(ctx)
	callCtx := llm.WithCallMeta(ctx, llm.CallMeta{Kind: llm.TaskComplex})
	resp, err := l.gateway.ChatStream(callCtx, out, nil, session.ID, step, onDelta)
	if err != nil {
		return ""
	}
	return readableAssistantText(resp.Content, resp.ReasoningContent)
}

func summarizeFromStepRecords(records []runtime.StepRecord) string {
	var tools []string
	var lastPlan string
	for _, r := range records {
		switch r.Kind {
		case "plan":
			lastPlan = r.Summary
		case "tool":
			line := r.ToolName
			if s := strings.TrimSpace(r.Summary); s != "" {
				line += ": " + s
			}
			tools = append(tools, line)
		}
	}
	if len(tools) == 0 && lastPlan == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("本回合因调用上限提前结束。已完成工作摘要：")
	if lastPlan != "" {
		b.WriteString("\n- 最近计划：")
		b.WriteString(lastPlan)
	}
	for _, t := range tools {
		b.WriteString("\n- ")
		b.WriteString(t)
	}
	return b.String()
}
