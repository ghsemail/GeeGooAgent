package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// withBudgetWarning appends a temporary user prompt when the turn is near the
// tool-round cap or context pressure is high. The warning is NOT persisted.
func withBudgetWarning(messages []llm.Message, round, maxRounds int, session *runtime.Session) []llm.Message {
	warn := budgetWarningText(round, maxRounds, session)
	if warn == "" {
		return messages
	}
	out := make([]llm.Message, len(messages)+1)
	copy(out, messages)
	out[len(messages)] = llm.Message{Role: llm.RoleUser, Content: warn}
	return out
}

func budgetWarningText(round, maxRounds int, session *runtime.Session) string {
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
	if session != nil && session.LastPromptTokens > 0 && session.LastPromptTokens >= 100000 {
		parts = append(parts, fmt.Sprintf(
			"[CONTEXT] 当前 prompt 约 %d tokens，请避免拉取冗长工具输出，必要时先总结再答。",
			session.LastPromptTokens,
		))
	}
	return strings.Join(parts, "\n")
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

func emptyReplyMessage(resp *llm.Response, records []runtime.StepRecord) string {
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

var coreSlimTools = []string{
	"search_code", "get_current_price", "get_ticker", "web_search",
	"get_mcp_analysis", "check_trading_day",
	"list_smart_trades", "list_dca_bots", "list_grid_bots", "list_hdg_bots",
	"fetch_market_news", "fetch_stock_news", "recall",
}

func slimSchemasForRetry(all []llm.ToolSchema, records []runtime.StepRecord) []llm.ToolSchema {
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
