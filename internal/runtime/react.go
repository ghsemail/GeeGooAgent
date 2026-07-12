package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const maxToolRounds = 8

// TurnResult is the outcome of one user turn.
type TurnResult struct {
	AssistantText string
	Failed        bool
	Error         string
	StepRecords   []StepRecord
}

// ReActLoop runs plan → act → observe for one chat turn.
type ReActLoop struct {
	gateway          *llm.Gateway
	executor         *Executor
	maxToolRounds    int
	onProgress       ProgressFunc
	compressor       *prompt.Compressor
	lastPromptTokens int
}

// NewReActLoop creates a ReAct loop.
func NewReActLoop(gateway *llm.Gateway, executor *Executor) *ReActLoop {
	return &ReActLoop{
		gateway:       gateway,
		executor:      executor,
		maxToolRounds: maxToolRounds,
	}
}

// SetGateway swaps the LLM gateway (e.g. after /think or /model).
func (l *ReActLoop) SetGateway(gateway *llm.Gateway) {
	l.gateway = gateway
}

// SetCompressor wires optional context compaction before LLM calls.
func (l *ReActLoop) SetCompressor(c *prompt.Compressor) {
	l.compressor = c
}

// SetProgress wires live step output (geegoo chat verbose UI).
func (l *ReActLoop) SetProgress(fn ProgressFunc) {
	l.onProgress = fn
}

func (l *ReActLoop) emit(event string, data map[string]any) {
	if l.onProgress != nil {
		l.onProgress(event, data)
	}
}

// RunTurn executes one user message through LLM + tools.
// ctx governs cancellation for the whole turn (LLM calls + tool execution).
func (l *ReActLoop) RunTurn(
	ctx context.Context,
	session *Session,
	userText string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
) TurnResult {
	if ctx == nil {
		ctx = context.Background()
	}
	toolCtx.Ctx = ctx
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: userText})
	messages := session.LLMMessages()
	records := []StepRecord{}

	l.emit("turn_start", map[string]any{"user_text": userText})

	for round := 0; round < l.maxToolRounds; round++ {
		if err := ctx.Err(); err != nil {
			msg := fmt.Sprintf("已中断: %v", err)
			l.emit("error", map[string]any{"message": msg})
			return TurnResult{AssistantText: msg, Failed: true, Error: err.Error(), StepRecords: records}
		}
		session.StepCounter++
		step := session.StepCounter
		l.emit("round_start", map[string]any{"round": round + 1, "step": step})

		messages = l.applyCompression(ctx, session, messages)
		resp, err := l.gateway.Chat(ctx, messages, schemas, session.ID, step)
		if err != nil {
			msg := fmt.Sprintf("模型调用失败: %v", err)
			l.emit("error", map[string]any{"message": msg})
			return TurnResult{AssistantText: msg, Failed: true, Error: err.Error(), StepRecords: records}
		}
		if resp.Usage.PromptTokens > 0 {
			l.lastPromptTokens = resp.Usage.PromptTokens
		}

		toolNames := make([]string, 0, len(resp.ToolCalls))
		for _, c := range resp.ToolCalls {
			toolNames = append(toolNames, c.Name)
		}
		planSummary := strings.TrimSpace(resp.Content)
		if planSummary == "" && strings.TrimSpace(resp.ReasoningContent) != "" {
			planSummary = strings.TrimSpace(resp.ReasoningContent)
		}
		if planSummary == "" && len(toolNames) > 0 {
			planSummary = fmt.Sprintf("决策: 调用 %s", strings.Join(toolNames, ", "))
		}
		if planSummary == "" {
			planSummary = "（无显式计划文本）"
		}
		if len(planSummary) > 300 {
			planSummary = planSummary[:300]
		}
		records = append(records, StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "plan", Summary: planSummary,
		})
		l.emit("llm_plan", map[string]any{
			"step": step, "content": resp.Content, "reasoning": resp.ReasoningContent, "tool_names": toolNames,
		})

		if len(resp.ToolCalls) == 0 {
			text := strings.TrimSpace(resp.Content)
			if text == "" {
				text = strings.TrimSpace(resp.ReasoningContent)
			}
			if text == "" {
				text = emptyReplyMessage(resp)
			}
			l.emit("reply_start", map[string]any{"step": step})
			session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: text})
			summary := text
			if len(summary) > 300 {
				summary = summary[:300]
			}
			records = append(records, StepRecord{
				Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: summary,
			})
			return TurnResult{AssistantText: text, StepRecords: records}
		}

		l.emit("llm_tools", map[string]any{"step": step, "tool_names": toolNames})

		assistant := llm.Message{
			Role: llm.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls,
			ReasoningContent: resp.ReasoningContent,
		}
		session.AppendMessage(assistant)
		messages = append(messages, assistant)

		for _, call := range resp.ToolCalls {
			toolCtx.Step = step
			l.emit("tool_start", map[string]any{
				"step": step, "name": call.Name, "arguments": call.Arguments,
			})
			result := l.executor.Execute(tools.CallRequest{
				Name: call.Name, Arguments: call.Arguments,
			}, toolCtx)
			l.emit("tool_done", map[string]any{
				"step": step, "name": call.Name, "status": string(result.Status),
				"summary": result.Summary, "arguments": call.Arguments,
			})
			summary := result.Summary
			if len(summary) > 300 {
				summary = summary[:300]
			}
			records = append(records, StepRecord{
				Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
				ToolName: call.Name, ToolStatus: string(result.Status), Summary: summary,
			})

			toolMsg := llm.Message{
				Role: llm.RoleTool, Content: toolResultContent(result), ToolCallID: call.ID,
			}
			session.AppendMessage(toolMsg)
			messages = append(messages, toolMsg)
		}
	}

	msg := "已达到单轮 Tool 调用上限，请缩小问题范围后重试。"
	l.emit("error", map[string]any{"message": msg})
	return TurnResult{AssistantText: msg, Failed: true, Error: "max_tool_rounds", StepRecords: records}
}

func (l *ReActLoop) applyCompression(ctx context.Context, session *Session, messages []llm.Message) []llm.Message {
	if l.compressor == nil {
		return messages
	}
	est := l.lastPromptTokens
	if est <= 0 {
		est = prompt.EstimateTokens(session.Messages)
	}
	if !l.compressor.ShouldCompress(est, len(session.Messages)) {
		return messages
	}
	before := len(session.Messages)
	out, did, newSummary, err := l.compressor.Compress(ctx, session.Messages, session.PreviousSummary, est)
	if err != nil || !did {
		return messages
	}
	session.Messages = out
	session.PreviousSummary = newSummary
	l.emit("context_compressed", map[string]any{
		"before_msgs":             before,
		"after_msgs":              len(out),
		"estimated_tokens_before": est,
		"summary_chars":           len(session.PreviousSummary),
	})
	return session.LLMMessages()
}

func toolResultContent(result tools.Result) string {
	payload := map[string]any{
		"status":  result.Status,
		"summary": result.Summary,
	}
	if result.Data != nil {
		payload["data"] = result.Data
	}
	raw, _ := json.Marshal(payload)
	text := string(raw)
	if len(text) > 6000 {
		return text[:6000]
	}
	return text
}

func emptyReplyMessage(resp *llm.Response) string {
	if resp != nil && strings.EqualFold(resp.FinishReason, "length") {
		return "模型输出被 max_tokens 截断（thinking 可能占满预算）。请提高 config.json 的 llm.max_tokens（建议 ≥8192），或 /think off / 降低 reasoning_effort 后重试。"
	}
	return "模型未返回可读内容。若开启了 thinking，请提高 llm.max_tokens 或执行 /think off 后重试。"
}
