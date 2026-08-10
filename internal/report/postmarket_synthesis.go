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

// PostMarketSynthesisResult is the LLM App-card summary bundle for postmarket.
type PostMarketSynthesisResult struct {
	MarketSummary     string `json:"market_summary"`
	TradeSummary      string `json:"trade_summary"`
	ExperienceSummary string `json:"experience_summary"`
	Summary           string `json:"summary"`
}

// SynthesizePostMarketSummaries writes App-facing postmarket summaries from the report draft.
func (s *Synthesizer) SynthesizePostMarketSummaries(
	ctx context.Context,
	ws memory.StockWorkspace,
	draft string,
	sessionBias, vsPreMarket string,
	ruleMarket, ruleTrade, ruleExperience string,
) (PostMarketSynthesisResult, error) {
	if !s.Available() {
		return PostMarketSynthesisResult{}, fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildPostMarketSynthesisPrompt(ws, draft, sessionBias, vsPreMarket, ruleMarket, ruleTrade, ruleExperience)
	callCtx := llm.WithCallMeta(cctx, llm.CallMeta{Kind: llm.TaskSynthesis})
	resp, err := s.gateway.Chat(callCtx, prompt, nil, "", 0)
	if err != nil {
		return PostMarketSynthesisResult{}, fmt.Errorf("postmarket synthesis LLM call: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return PostMarketSynthesisResult{}, fmt.Errorf("postmarket synthesis returned empty content")
	}
	parsed, err := parsePostMarketSynthesisJSON(content)
	if err != nil {
		return PostMarketSynthesisResult{}, fmt.Errorf("postmarket synthesis parse: %w", err)
	}
	parsed.MarketSummary = cleanPostSummary(parsed.MarketSummary)
	parsed.TradeSummary = cleanPostSummary(parsed.TradeSummary)
	parsed.ExperienceSummary = cleanPostSummary(parsed.ExperienceSummary)
	parsed.Summary = cleanPostSummary(parsed.Summary)
	if parsed.MarketSummary == "" || parsed.TradeSummary == "" || parsed.ExperienceSummary == "" || parsed.Summary == "" {
		return PostMarketSynthesisResult{}, fmt.Errorf("postmarket synthesis missing summary fields")
	}
	if n := len([]rune(parsed.Summary)); n > 220 {
		return PostMarketSynthesisResult{}, fmt.Errorf("postmarket synthesis summary too long (%d chars)", n)
	}
	return parsed, nil
}

func cleanPostSummary(s string) string {
	s = strings.TrimSpace(stockfmt.LocalizeDecisionTerms(stockfmt.StripEvidenceRefs(s)))
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}

func buildPostMarketSynthesisPrompt(
	ws memory.StockWorkspace,
	draft, sessionBias, vsPreMarket, ruleMarket, ruleTrade, ruleExperience string,
) []llm.Message {
	var b strings.Builder
	b.WriteString("你是个股盘后报告摘要器。只能依据下列草稿与事实写摘要，禁止编造未给出的成交、价格或盘前结论。\n\n")
	b.WriteString(fmt.Sprintf("标的: %s (%s)\n", ws.StockName, ws.Code))
	b.WriteString(fmt.Sprintf("Bot: %s（%s）\n", ws.BotName, ws.BotType))
	b.WriteString(fmt.Sprintf("涨跌幅: %.2f%%\n", ws.ChangePct))
	b.WriteString(fmt.Sprintf("盘面倾向: %s\n", sessionBias))
	b.WriteString(fmt.Sprintf("与盘前对照: %s\n", vsPreMarket))
	if strings.TrimSpace(ws.PreMarketResult) != "" {
		b.WriteString(fmt.Sprintf("盘前判断: %s\n", ws.PreMarketResult))
	} else {
		b.WriteString("盘前判断: 无\n")
	}
	if strings.TrimSpace(ws.BotLogSummary) != "" {
		b.WriteString("Bot 日志摘要: " + truncateLine(ws.BotLogSummary, 600) + "\n")
	}
	b.WriteString("\n规则引擎参考摘要（可改写润色，勿照抄套话）:\n")
	b.WriteString("- market_summary: " + nonEmpty(ruleMarket, "(无)") + "\n")
	b.WriteString("- trade_summary: " + nonEmpty(ruleTrade, "(无)") + "\n")
	b.WriteString("- experience_summary: " + nonEmpty(ruleExperience, "(无)") + "\n\n")
	b.WriteString("盘后报告草稿:\n")
	b.WriteString(nonEmpty(truncateLine(draft, 12000), "(无草稿)"))
	b.WriteString(`

要求:
- 四个字段均为中文纯文本，禁止 Markdown、表格、[ev_...]、英文枚举原词
- market_summary: 浓缩今日走势与小时级要点，须含涨跌方向/幅度与关键观察，约 80～220 字
- trade_summary: 浓缩当日 Bot 成交与持仓/策略执行情况；无成交也要说明，约 40～160 字
- experience_summary: 写可复用的复盘教训（对照盘前、执行偏差、次日关注点），约 80～220 字；不得与 market_summary 雷同
- summary: 一句话总览，<=200 字
- 输出严格 JSON: {"market_summary":"...","trade_summary":"...","experience_summary":"...","summary":"..."}`)
	return []llm.Message{{Role: "user", Content: b.String()}}
}

func parsePostMarketSynthesisJSON(content string) (PostMarketSynthesisResult, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var out PostMarketSynthesisResult
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
