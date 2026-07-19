package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (l *Loop) resumePendingPlan(
	ctx context.Context,
	session *runtime.Session,
	messages *[]llm.Message,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
	records *[]runtime.StepRecord,
) runtime.TurnResult {
	plan := session.PendingPlan
	if plan == nil || len(plan.ToolCalls) == 0 {
		session.PendingPlan = nil
		return runtime.TurnResult{AssistantText: "无待确认的写操作。", StepRecords: *records}
	}
	session.PendingPlan = nil
	toolCtx.Approved = true

	l.emit("plan_confirmed", map[string]any{"tools": mutatingToolNamesFromCalls(plan.ToolCalls)})

	results := l.executeToolCalls(ctx, plan.ToolCalls, toolCtx, plan.Step)
	for i, call := range plan.ToolCalls {
		result := results[i]
		summary := result.Summary
		if len(summary) > 300 {
			summary = summary[:300]
		}
		*records = append(*records, runtime.StepRecord{
			Step: plan.Step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: call.Name, ToolStatus: string(result.Status), Summary: summary,
		})
		toolMsg := llm.Message{
			Role: llm.RoleTool, Content: toolResultContent(result), ToolCallID: call.ID,
		}
		session.AppendMessage(toolMsg)
		*messages = append(*messages, toolMsg)
	}

	for round := 0; round < l.maxToolRounds; round++ {
		if err := ctx.Err(); err != nil {
			return l.failTurn(ctx, session, err, *records)
		}
		done, result := l.runRound(ctx, session, messages, toolCtx, schemas, round, records)
		if done {
			if !result.Failed {
				l.emitBus("TurnCompleted", map[string]any{
					"session_id": session.ID, "steps": len(result.StepRecords),
				})
			}
			return result
		}
	}
	return l.finishBudgetExhausted(ctx, session, *messages, *records)
}

func mutatingToolNamesFromCalls(calls []llm.ToolCall) []string {
	names := make([]string, 0, len(calls))
	for _, c := range calls {
		if tools.ApprovalRequired(c.Name) {
			names = append(names, c.Name)
		}
	}
	return names
}

func (l *Loop) cancelPendingPlan(session *runtime.Session) runtime.TurnResult {
	session.PendingPlan = nil
	return runtime.TurnResult{AssistantText: "已取消计划中的写操作。", PlanPending: false}
}

func appendToolResults(
	session *runtime.Session,
	messages *[]llm.Message,
	calls []llm.ToolCall,
	results []tools.Result,
	step int,
	records *[]runtime.StepRecord,
) {
	for i, call := range calls {
		result := results[i]
		summary := result.Summary
		if len(summary) > 300 {
			summary = summary[:300]
		}
		*records = append(*records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: call.Name, ToolStatus: string(result.Status), Summary: summary,
		})
		toolMsg := llm.Message{
			Role: llm.RoleTool, Content: toolResultContent(result), ToolCallID: call.ID,
		}
		session.AppendMessage(toolMsg)
		*messages = append(*messages, toolMsg)
	}
}

func planHoldSummary(policy cognition.PlanPolicy, resp *llm.Response, mutating []llm.ToolCall) string {
	names := make([]string, 0, len(mutating))
	for _, c := range mutating {
		names = append(names, c.Name)
	}
	text := readableAssistantText(resp.Content, resp.ReasoningContent)
	if text == "" {
		text = fmt.Sprintf("计划调用写操作：%s", strings.Join(names, ", "))
	}
	return planHoldUserMessage(policy, text)
}
