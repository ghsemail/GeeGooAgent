package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// IntradaySynthesisResult is LLM-generated summary and reason for intraday reports.
type IntradaySynthesisResult struct {
	Summary string `json:"summary"`
	Reason  string `json:"reason"`
}

// SynthesizeIntraday writes user-facing summary and reason from a formatted draft report.
func (s *Synthesizer) SynthesizeIntraday(
	ctx context.Context,
	ws memory.StockWorkspace,
	draft string,
	result, confidence string,
) (IntradaySynthesisResult, error) {
	if !s.Available() {
		return IntradaySynthesisResult{}, fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildIntradaySynthesisPrompt(ws, draft, result, confidence)
	callCtx := llm.WithCallMeta(cctx, llm.CallMeta{Kind: llm.TaskSynthesis})
	resp, err := s.gateway.Chat(callCtx, prompt, nil, "", 0)
	if err != nil {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis LLM call: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis returned empty content")
	}
	parsed, err := parseIntradaySynthesisJSON(content)
	if err != nil {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis parse: %w", err)
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis summary empty")
	}
	if strings.TrimSpace(parsed.Reason) == "" {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis reason empty")
	}
	if len([]rune(parsed.Reason)) < 80 {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis reason too short (%d chars)", len([]rune(parsed.Reason)))
	}
	if len([]rune(parsed.Summary)) > 220 {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis summary too long (%d chars)", len([]rune(parsed.Summary)))
	}
	parsed.Summary = stockfmt.LocalizeDecisionTerms(stockfmt.StripEvidenceRefs(parsed.Summary))
	parsed.Reason = stockfmt.LocalizeDecisionTerms(stockfmt.StripEvidenceRefs(parsed.Reason))
	return parsed, nil
}

func buildIntradaySynthesisPrompt(ws memory.StockWorkspace, draft, result, confidence string) []llm.Message {
	var b strings.Builder
	b.WriteString("你是个股盘中决策报告综合器。只能引用下面草稿与字段，禁止编造未给出的价格或指标。\n\n")
	b.WriteString(fmt.Sprintf("标的: %s (%s)\n", ws.StockName, ws.Code))
	b.WriteString(fmt.Sprintf("Bot: %s (%s)\n", ws.BotName, ws.BotType))
	b.WriteString(fmt.Sprintf("本轮信号: %s\n", ws.TradeType))
	b.WriteString(fmt.Sprintf("规则决策: %s（置信度 %s）\n\n", result, confidence))
	b.WriteString("已排版报告草稿:\n")
	b.WriteString(nonEmpty(draft, "(无草稿)"))
	b.WriteString(`

要求:
- summary: <=200字，完整的一句话或两句中文结论；禁止 Markdown、标签、英文枚举（buy/sell/hold/high 等须写中文）
- reason: >=80字，自然中文说明判定依据；禁止 Markdown 表格、参数名、科学计数法、[ev_...] 标签
- 不要重复 App 概要区已有的「决策/置信度」标签字样，直接写分析结论
- 输出严格 JSON: {"summary":"...","reason":"..."}`)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: "你是严格的 JSON 报告综合器, 只输出 JSON, 不输出任何其它文字。"},
		{Role: llm.RoleUser, Content: b.String()},
	}
}

func parseIntradaySynthesisJSON(content string) (IntradaySynthesisResult, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var out IntradaySynthesisResult
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(content[start:end+1]), &out); err2 == nil {
				return out, nil
			}
		}
		return out, err
	}
	return out, nil
}
