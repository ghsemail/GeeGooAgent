package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const defaultMaxToolRounds = 80

// TurnResult is the outcome of one user turn.
type TurnResult struct {
	AssistantText string
	Failed        bool
	Error         string
	StepRecords   []StepRecord
}

// ReActLoop runs plan → act → observe for one chat turn.
type ReActLoop struct {
	gateway       *llm.Gateway
	executor      *Executor
	maxToolRounds int
	onProgress    ProgressFunc
	approvalFn    ApprovalFunc
	compressor    *prompt.Compressor
}

// NewReActLoop creates a ReAct loop.
func NewReActLoop(gateway *llm.Gateway, executor *Executor) *ReActLoop {
	return &ReActLoop{
		gateway:       gateway,
		executor:      executor,
		maxToolRounds: defaultMaxToolRounds,
	}
}

// SetMaxToolRounds sets the per-turn LLM↔tool iteration cap (config max_steps).
func (l *ReActLoop) SetMaxToolRounds(n int) {
	if l == nil {
		return
	}
	if n <= 0 {
		n = defaultMaxToolRounds
	}
	if n > 90 {
		n = 90
	}
	l.maxToolRounds = n
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

// SetApproval wires interactive confirmation for mutating tools.
func (l *ReActLoop) SetApproval(fn ApprovalFunc) {
	l.approvalFn = fn
}

func (l *ReActLoop) emit(event string, data map[string]any) {
	if l.onProgress != nil {
		l.onProgress(event, data)
	}
}

func (l *ReActLoop) streamHandler(ctx context.Context) llm.StreamHandler {
	outer := llm.StreamHandlerFrom(ctx)
	if outer == nil && l.onProgress == nil {
		return nil
	}
	return func(delta llm.StreamDelta) {
		if delta.Content != "" {
			l.emit("stream_delta", map[string]any{"content": delta.Content})
		}
		if delta.ReasoningContent != "" {
			l.emit("stream_delta", map[string]any{"reasoning": delta.ReasoningContent})
		}
		if outer != nil {
			outer(delta)
		}
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
	messages = l.applyHygiene(ctx, session, messages)

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
		apiMessages := withBudgetWarning(messages, round, l.maxToolRounds, session)
		if len(apiMessages) > len(messages) {
			l.emit("budget_warning", map[string]any{
				"round": round + 1, "max_rounds": l.maxToolRounds,
				"remaining": l.maxToolRounds - round,
			})
		}
		onDelta := l.streamHandler(ctx)
		resp, err := l.gateway.ChatStream(ctx, apiMessages, schemas, session.ID, step, onDelta)
		if err != nil {
			msg := fmt.Sprintf("模型调用失败: %v", err)
			l.emit("error", map[string]any{"message": msg})
			return TurnResult{AssistantText: msg, Failed: true, Error: err.Error(), StepRecords: records}
		}
		if llm.MalformedToolCallResponse(resp) {
			if slim := slimSchemasForRetry(schemas, records); len(slim) > 0 && len(slim) < len(schemas) {
				l.emit("llm_tools_slim_retry", map[string]any{"from": len(schemas), "to": len(slim)})
				resp2, err2 := l.gateway.ChatStream(ctx, apiMessages, slim, session.ID, step, onDelta)
				if err2 != nil {
					msg := fmt.Sprintf("模型调用失败: %v", err2)
					l.emit("error", map[string]any{"message": msg})
					return TurnResult{AssistantText: msg, Failed: true, Error: err2.Error(), StepRecords: records}
				}
				resp = resp2
			}
		}
		if resp.Usage.PromptTokens > 0 {
			session.LastPromptTokens = resp.Usage.PromptTokens
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

		toolResults := l.executeToolCalls(ctx, resp.ToolCalls, toolCtx, step)
		for i, call := range resp.ToolCalls {
			result := toolResults[i]
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

func (l *ReActLoop) executeToolCalls(
	ctx context.Context,
	calls []llm.ToolCall,
	toolCtx tools.Context,
	step int,
) []tools.Result {
	results := make([]tools.Result, len(calls))
	if len(calls) == 0 {
		return results
	}
	runOne := func(i int, call llm.ToolCall) {
		tc := toolCtx
		tc.Step = step
		if tc.Interactive && tools.ApprovalRequired(call.Name) && !tc.Approved {
			approved := false
			if l.approvalFn != nil {
				approved = l.approvalFn(call.Name, call.Arguments)
			}
			if !approved {
				result := tools.Result{
					Status:  tools.StatusSkip,
					Summary: "需要确认：" + call.Name + " 是写操作，请确认后再执行",
					Data:    map[string]any{"tool": call.Name, "approval_required": true},
				}
				l.emit("tool_done", map[string]any{
					"step": step, "name": call.Name, "status": string(result.Status),
					"summary": result.Summary, "arguments": call.Arguments,
				})
				results[i] = result
				return
			}
			tc.Approved = true
		}
		l.emit("tool_start", map[string]any{
			"step": step, "name": call.Name, "arguments": call.Arguments,
		})
		result := l.executor.Execute(tools.CallRequest{
			Name: call.Name, Arguments: call.Arguments,
		}, tc)
		l.emit("tool_done", map[string]any{
			"step": step, "name": call.Name, "status": string(result.Status),
			"summary": result.Summary, "arguments": call.Arguments,
		})
		results[i] = result
	}
	if len(calls) == 1 || needsInteractiveApproval(toolCtx, calls) {
		for i, call := range calls {
			if err := ctx.Err(); err != nil {
				for j := i; j < len(calls); j++ {
					results[j] = tools.Result{
						Status:  tools.StatusError,
						Summary: fmt.Sprintf("已中断: %v", err),
					}
				}
				return results
			}
			runOne(i, call)
		}
		return results
	}
	var wg sync.WaitGroup
	for i, call := range calls {
		if err := ctx.Err(); err != nil {
			for j := i; j < len(calls); j++ {
				results[j] = tools.Result{
					Status:  tools.StatusError,
					Summary: fmt.Sprintf("已中断: %v", err),
				}
			}
			return results
		}
		wg.Add(1)
		go func(i int, call llm.ToolCall) {
			defer wg.Done()
			runOne(i, call)
		}(i, call)
	}
	wg.Wait()
	return results
}

func needsInteractiveApproval(toolCtx tools.Context, calls []llm.ToolCall) bool {
	if !toolCtx.Interactive {
		return false
	}
	for _, call := range calls {
		if tools.ApprovalRequired(call.Name) {
			return true
		}
	}
	return false
}

// withBudgetWarning appends a temporary user prompt (Hermes-style) when the
// turn is near the tool-round cap or context pressure is high. The warning is
// NOT persisted into session history — only the outbound API payload.
func withBudgetWarning(messages []llm.Message, round, maxRounds int, session *Session) []llm.Message {
	warn := budgetWarningText(round, maxRounds, session)
	if warn == "" {
		return messages
	}
	out := make([]llm.Message, len(messages)+1)
	copy(out, messages)
	out[len(messages)] = llm.Message{Role: llm.RoleUser, Content: warn}
	return out
}

func budgetWarningText(round, maxRounds int, session *Session) string {
	if maxRounds <= 0 {
		return ""
	}
	remaining := maxRounds - round
	var parts []string
	if remaining <= 2 {
		parts = append(parts, fmt.Sprintf(
			"[BUDGET] 本回合还剩 %d/%d 次模型调用。请尽快给出最终答复，避免继续无必要的工具调用。",
			remaining, maxRounds,
		))
	} else if remaining <= maxRounds/5 && maxRounds >= 10 {
		parts = append(parts, fmt.Sprintf(
			"[BUDGET] 已使用 %d/%d 次模型调用。请优先收敛结论。",
			round, maxRounds,
		))
	}
	if session != nil && session.LastPromptTokens > 0 {
		// Soft context-pressure hint once tokens look large (≈100k+).
		if session.LastPromptTokens >= 100000 {
			parts = append(parts, fmt.Sprintf(
				"[CONTEXT] 当前 prompt 约 %d tokens，请避免拉取冗长工具输出，必要时先总结再答。",
				session.LastPromptTokens,
			))
		}
	}
	return strings.Join(parts, "\n")
}

func (l *ReActLoop) applyCompression(ctx context.Context, session *Session, messages []llm.Message) []llm.Message {
	return l.runCompression(ctx, session, messages, false)
}

func (l *ReActLoop) applyHygiene(ctx context.Context, session *Session, messages []llm.Message) []llm.Message {
	return l.runCompression(ctx, session, messages, true)
}

func (l *ReActLoop) runCompression(ctx context.Context, session *Session, messages []llm.Message, hygiene bool) []llm.Message {
	if l.compressor == nil {
		return messages
	}
	est := session.LastPromptTokens
	if est <= 0 {
		est = prompt.EstimateTokens(session.Messages)
	}
	var (
		out        []llm.Message
		did        bool
		newSummary string
		err        error
	)
	if hygiene {
		if !l.compressor.ShouldHygiene(est, len(session.Messages)) {
			return messages
		}
		out, did, newSummary, err = l.compressor.CompressHygiene(ctx, session.Messages, session.PreviousSummary, est)
	} else {
		if !l.compressor.ShouldCompress(est, len(session.Messages)) {
			return messages
		}
		out, did, newSummary, err = l.compressor.Compress(ctx, session.Messages, session.PreviousSummary, est)
	}
	if err != nil || !did {
		return messages
	}
	before := len(session.Messages)
	session.Messages = out
	session.PreviousSummary = newSummary
	session.LastPromptTokens = prompt.EstimateTokens(out)
	advanceLineage(session)
	event := "context_compressed"
	if hygiene {
		event = "context_hygiene"
	}
	l.emit(event, map[string]any{
		"before_msgs":             before,
		"after_msgs":              len(out),
		"estimated_tokens_before": est,
		"summary_chars":           len(session.PreviousSummary),
		"hygiene":                 hygiene,
		"parent_id":               session.ParentID,
		"lineage_root":            session.LineageRoot,
		"compaction_generation":   session.CompactionGeneration,
	})
	return session.LLMMessages()
}

// advanceLineage records a Hermes-style child generation after compaction.
// The user-facing session.ID stays stable; ParentID points at the prior node.
func advanceLineage(session *Session) {
	if session == nil {
		return
	}
	if session.LineageRoot == "" {
		session.LineageRoot = session.ID
	}
	if session.CompactionGeneration <= 0 {
		session.ParentID = session.LineageRoot
	} else {
		session.ParentID = fmt.Sprintf("%s#g%d", session.LineageRoot, session.CompactionGeneration)
	}
	session.CompactionGeneration++
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
