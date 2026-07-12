package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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
		if llm.MalformedToolCallResponse(resp) {
			if slim := slimSchemasForRetry(schemas, records); len(slim) > 0 && len(slim) < len(schemas) {
				l.emit("llm_tools_slim_retry", map[string]any{"from": len(schemas), "to": len(slim)})
				resp2, err2 := l.gateway.Chat(ctx, messages, slim, session.ID, step)
				if err2 != nil {
					msg := fmt.Sprintf("模型调用失败: %v", err2)
					l.emit("error", map[string]any{"message": msg})
					return TurnResult{AssistantText: msg, Failed: true, Error: err2.Error(), StepRecords: records}
				}
				resp = resp2
			}
		}
		if resp.Usage.PromptTokens > 0 {
			l.lastPromptTokens = resp.Usage.PromptTokens
		}

		toolNames := make([]string, 0, len(resp.ToolCalls))
		for _, c := range resp.ToolCalls {
			toolNames = append(toolNames, c.Name)
		}
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
		records = append(records, StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "plan", Summary: planSummary,
		})
		l.emit("llm_plan", map[string]any{
			"step": step, "content": resp.Content, "reasoning": resp.ReasoningContent, "tool_names": toolNames,
		})

		if len(resp.ToolCalls) == 0 {
			text := readableAssistantText(resp.Content, resp.ReasoningContent)
			if text == "" {
				text = emptyReplyMessage(resp, records)
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

func emptyReplyMessage(resp *llm.Response, records []StepRecord) string {
	var fails []string
	for _, r := range records {
		if r.Kind != "tool" {
			continue
		}
		if !strings.EqualFold(r.ToolStatus, "error") {
			continue
		}
		line := r.ToolName
		if s := strings.TrimSpace(r.Summary); s != "" {
			line += ": " + s
		}
		fails = append(fails, line)
	}
	if len(fails) > 0 {
		var b strings.Builder
		b.WriteString("工具调用失败，且模型未返回可读说明：")
		for _, f := range fails {
			b.WriteString("\n- ")
			b.WriteString(f)
		}
		return b.String()
	}
	if resp != nil && llm.MalformedToolCallResponse(resp) {
		return "模型声称要调用工具但未返回 tool_calls（网关异常）。请重试；若反复出现，可更换模型或减少启用工具。"
	}
	if resp != nil && strings.EqualFold(resp.FinishReason, "length") {
		return "模型输出被 max_tokens 截断（thinking 可能占满预算）。请提高 config.json 的 llm.max_tokens（建议 ≥8192），或 /think off / 降低 reasoning_effort 后重试。"
	}
	return "模型未返回可读内容。若开启了 thinking，请提高 llm.max_tokens 或执行 /think off 后重试。"
}

// readableAssistantText prefers content, then reasoning, and strips provider noise like [SID=...].
func readableAssistantText(content, reasoning string) string {
	if text := stripProviderNoise(content); text != "" {
		return text
	}
	return stripProviderNoise(reasoning)
}

var sidTokenRE = regexp.MustCompile(`(?i)\[SID=[^\]]+\]`)

func stripProviderNoise(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.TrimSpace(sidTokenRE.ReplaceAllString(s, ""))
}

// coreSlimTools is the fallback allowlist when the provider returns
// finish_reason=tool_calls without a tool_calls payload under a large catalog.
var coreSlimTools = []string{
	"search_code", "get_current_price", "get_ticker", "web_search",
	"get_mcp_analysis", "check_trading_day",
	"list_smart_trades", "list_dca_bots", "list_grid_bots", "list_hdg_bots",
	"fetch_market_news", "fetch_stock_news", "recall",
}

func slimSchemasForRetry(all []llm.ToolSchema, records []StepRecord) []llm.ToolSchema {
	want := make(map[string]struct{}, len(coreSlimTools)+4)
	for _, name := range coreSlimTools {
		want[name] = struct{}{}
	}
	for _, r := range records {
		if r.Kind == "tool" && strings.TrimSpace(r.ToolName) != "" {
			want[r.ToolName] = struct{}{}
		}
	}
	out := make([]llm.ToolSchema, 0, len(want))
	for _, schema := range all {
		if _, ok := want[schema.Name]; ok {
			out = append(out, schema)
		}
	}
	return out
}
