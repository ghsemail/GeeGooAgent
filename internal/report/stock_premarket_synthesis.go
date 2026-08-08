package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// StockPreMarketSynthesisResult is the LLM-generated stock pre-market bundle.
type StockPreMarketSynthesisResult struct {
	Report     string `json:"report"`
	Result     string `json:"result"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
	Suggestion string `json:"suggestion"`
	Summary    string `json:"summary"`
}

// SynthesizeStockPreMarket writes a stock pre-market report from evidence and draft.
func (s *Synthesizer) SynthesizeStockPreMarket(
	ctx context.Context,
	ws memory.StockWorkspace,
	draft string,
	evidence []memory.EvidenceRef,
	marketContext memory.MarketContext,
	marketReportSummary string,
	template string,
) (StockPreMarketSynthesisResult, error) {
	if !s.Available() {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildStockPreMarketSynthesisPrompt(ws, draft, evidence, marketContext, marketReportSummary, template)
	callCtx := llm.WithCallMeta(cctx, llm.CallMeta{Kind: llm.TaskSynthesis})
	resp, err := s.gateway.Chat(callCtx, prompt, nil, "", 0)
	if err != nil {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis LLM call: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis returned empty content")
	}
	parsed, err := parseStockPreMarketSynthesisJSON(content)
	if err != nil {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis parse: %w", err)
	}
	if strings.TrimSpace(parsed.Report) == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis report empty")
	}
	if strings.TrimSpace(parsed.Reason) == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis reason empty")
	}
	if len([]rune(parsed.Reason)) < 80 {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis reason too short (%d chars)", len([]rune(parsed.Reason)))
	}
	parsed.Result = normalizeMarketResult(parsed.Result)
	parsed.Confidence = normalizeMarketConfidence(parsed.Confidence)
	parsed.Suggestion = normalizeStockSuggestion(parsed.Suggestion)
	if parsed.Result == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis result invalid")
	}
	if parsed.Confidence == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis confidence invalid")
	}
	if parsed.Suggestion == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis suggestion invalid")
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis summary empty")
	}
	if len([]rune(parsed.Summary)) > 220 {
		return StockPreMarketSynthesisResult{}, fmt.Errorf("stock premarket synthesis summary too long (%d chars)", len([]rune(parsed.Summary)))
	}
	return parsed, nil
}

func buildStockPreMarketSynthesisPrompt(
	ws memory.StockWorkspace,
	draft string,
	evidence []memory.EvidenceRef,
	mc memory.MarketContext,
	marketReportSummary, template string,
) []llm.Message {
	var b strings.Builder
	b.WriteString("你是个股盘前报告综合器。只能引用下面提供的证据与草稿，禁止编造价格、态度、资金流或任何未给出的数据。\n\n")
	b.WriteString(fmt.Sprintf("标的: %s (%s)\n", ws.StockName, ws.Code))
	b.WriteString(fmt.Sprintf("Bot: %s (%s, id=%s)\n", ws.BotName, ws.BotType, ws.BotID))
	b.WriteString(fmt.Sprintf("Bot 昨日态度: %s\n\n", nonEmpty(ws.Attitude, "neutral")))
	if strings.TrimSpace(template) != "" {
		b.WriteString("报告模板（遵循章节结构，勿输出模板注释）:\n")
		b.WriteString(template)
		b.WriteString("\n\n")
	}
	b.WriteString("规则草稿（可改写为更通顺的中文报告，但不得添加新事实）:\n")
	b.WriteString(nonEmpty(draft, "(无草稿)"))
	b.WriteString("\n\n当日市场盘前摘要:\n")
	b.WriteString(nonEmpty(marketReportSummary, "(无市场盘前摘要)"))
	b.WriteString("\n\n证据 refs:\n")
	for _, ev := range evidence {
		b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", ev.ID, ev.Source, ev.Summary))
	}
	if len(evidence) == 0 {
		b.WriteString("- (无 evidence ref，仅使用上文数据)\n")
	}
	b.WriteString("\n周线技术分析: " + nonEmpty(ws.WeeklyAnalysisRef, "(未捕获)") + "\n")
	b.WriteString("资金流: " + nonEmpty(ws.CapitalFlowSummary, "(未捕获)") + "\n")
	b.WriteString("资金分布: " + nonEmpty(ws.CapitalDistributionSummary, "(未捕获)") + "\n")
	b.WriteString("个股新闻: " + nonEmpty(ws.StockNewsSummary, "(未捕获)") + "\n")
	b.WriteString(`
要求:
- report: 完整 Markdown，遵循模板章节；禁止 Markdown 表格；禁止 # 标题行与元数据
- 正文不要重复 result/confidence/suggestion（由 API 字段与 App 概要区展示）
- 市场背景只写摘要，不要粘贴整份市场报告
- 个股新闻 3–5 条要点，禁止罗列发布时间或 🕐 时间戳
- result: long / short / neutral；confidence: high / medium / low；suggestion: buy / sell / hold
- reason: >=80字，必须引用具体证据，禁止空洞表述
- summary: <=200字，面向用户的一句话结论
- 正文末尾仅一行脚注：*报告由 GeeGoo 智能体个股盘前 skill 生成*
- 输出严格 JSON: {"report":"...","result":"...","confidence":"...","reason":"...","suggestion":"...","summary":"..."}`)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: "你是严格的 JSON 个股报告综合器, 只输出 JSON, 不输出任何其它文字。"},
		{Role: llm.RoleUser, Content: b.String()},
	}
}

func parseStockPreMarketSynthesisJSON(content string) (StockPreMarketSynthesisResult, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var out StockPreMarketSynthesisResult
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

func normalizeStockSuggestion(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "buy", "long", "bullish":
		return "buy"
	case "sell", "short", "bearish", "reduce_or_avoid":
		return "sell"
	case "hold", "neutral", "watch", "watch_long":
		return "hold"
	default:
		return ""
	}
}
