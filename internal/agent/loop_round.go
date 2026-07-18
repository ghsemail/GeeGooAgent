package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// runRound executes one LLM↔tool iteration. The second return is true when the turn ends.
func (l *Loop) runRound(
	ctx context.Context,
	session *runtime.Session,
	messages *[]llm.Message,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
	round int,
	records *[]runtime.StepRecord,
) (bool, runtime.TurnResult) {
	session.StepCounter++
	step := session.StepCounter
	l.emit("round_start", map[string]any{"round": round + 1, "step": step})

	*messages = l.applyCompression(ctx, session, *messages)
	apiMessages := withBudgetWarning(*messages, round, l.maxToolRounds, session)
	apiMessages = llm.SanitizeMessages(apiMessages)
	if len(apiMessages) > len(*messages) {
		l.emit("budget_warning", map[string]any{
			"round": round + 1, "max_rounds": l.maxToolRounds,
			"remaining": l.maxToolRounds - round,
		})
	}

	resp, err := l.callLLM(ctx, apiMessages, schemas, session.ID, step, records)
	if err != nil {
		msg := fmt.Sprintf("模型调用失败: %v", err)
		l.emit("error", map[string]any{"message": msg})
		l.emitBus("TurnFailed", map[string]any{
			"session_id": session.ID, "error": err.Error(),
		})
		return true, runtime.TurnResult{AssistantText: msg, Failed: true, Error: err.Error(), StepRecords: *records}
	}
	if resp.Usage.PromptTokens > 0 {
		session.LastPromptTokens = resp.Usage.PromptTokens
	}

	toolNames := toolCallNames(resp.ToolCalls)
	planSummary := planSummaryText(resp, toolNames)
	*records = append(*records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "plan", Summary: planSummary,
	})
	l.emit("llm_plan", map[string]any{
		"step": step, "content": resp.Content, "reasoning": resp.ReasoningContent, "tool_names": toolNames,
	})

	if len(resp.ToolCalls) == 0 {
		l.emitStepComplete(step, round, false, nil)
		return true, l.finalizeReply(session, resp, step, records)
	}

	l.emitStepComplete(step, round, true, toolNames)
	return false, l.applyToolRound(ctx, session, messages, toolCtx, resp, step, toolNames, records)
}

func (l *Loop) callLLM(
	ctx context.Context,
	messages []llm.Message,
	schemas []llm.ToolSchema,
	sessionID string,
	step int,
	records *[]runtime.StepRecord,
) (*llm.Response, error) {
	onDelta := l.streamHandler(ctx)
	resp, err := l.gateway.ChatStream(ctx, messages, schemas, sessionID, step, onDelta)
	if err != nil {
		return nil, err
	}
	if !llm.MalformedToolCallResponse(resp) {
		return resp, nil
	}
	slim := slimSchemasForRetry(schemas, *records)
	if len(slim) == 0 || len(slim) >= len(schemas) {
		return resp, nil
	}
	l.emit("llm_tools_slim_retry", map[string]any{"from": len(schemas), "to": len(slim)})
	return l.gateway.ChatStream(ctx, messages, slim, sessionID, step, onDelta)
}

func (l *Loop) finalizeReply(
	session *runtime.Session,
	resp *llm.Response,
	step int,
	records *[]runtime.StepRecord,
) runtime.TurnResult {
	text := readableAssistantText(resp.Content, resp.ReasoningContent)
	if text == "" {
		text = emptyReplyMessage(resp, *records)
	}
	l.emit("reply_start", map[string]any{"step": step})
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: text})
	summary := text
	if len(summary) > 300 {
		summary = summary[:300]
	}
	*records = append(*records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: summary,
	})
	return runtime.TurnResult{AssistantText: text, StepRecords: *records}
}

func (l *Loop) applyToolRound(
	ctx context.Context,
	session *runtime.Session,
	messages *[]llm.Message,
	toolCtx tools.Context,
	resp *llm.Response,
	step int,
	toolNames []string,
	records *[]runtime.StepRecord,
) runtime.TurnResult {
	l.emit("llm_tools", map[string]any{"step": step, "tool_names": toolNames})

	assistant := llm.Message{
		Role: llm.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls,
		ReasoningContent: resp.ReasoningContent,
	}
	session.AppendMessage(assistant)
	*messages = append(*messages, assistant)

	toolResults := l.executeToolCalls(ctx, resp.ToolCalls, toolCtx, step)
	for i, call := range resp.ToolCalls {
		result := toolResults[i]
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
	return runtime.TurnResult{}
}

func toolCallNames(calls []llm.ToolCall) []string {
	names := make([]string, 0, len(calls))
	for _, c := range calls {
		names = append(names, c.Name)
	}
	return names
}

func planSummaryText(resp *llm.Response, toolNames []string) string {
	planSummary := readableAssistantText(resp.Content, resp.ReasoningContent)
	if planSummary == "" && len(toolNames) > 0 {
		planSummary = fmt.Sprintf("决策: 调用 %s", strings.Join(toolNames, ", "))
	}
	if planSummary == "" {
		planSummary = "（无显式计划文本）"
	}
	if len(planSummary) > 300 {
		planSummary = planSummary[:300]
	}
	return planSummary
}
