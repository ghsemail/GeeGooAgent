package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// MarketSynthesisResult is the LLM-generated market pre-market report bundle.
type MarketSynthesisResult struct {
	Report     string `json:"report"`
	Result     string `json:"result"`
	Confidence string `json:"confidence"`
	Summary    string `json:"summary"`
}

// SynthesizeMarket asks the LLM to write a single-market pre-market report from
// captured index/news evidence. Callers must treat errors as fatal (no rule-based fallback).
func (s *Synthesizer) SynthesizeMarket(
	ctx context.Context,
	market string,
	draft string,
	marketContext memory.MarketContext,
	evidence []memory.EvidenceRef,
	template string,
) (MarketSynthesisResult, error) {
	if !s.Available() {
		return MarketSynthesisResult{}, fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildMarketSynthesisPrompt(market, draft, marketContext, evidence, template)
	var parsed MarketSynthesisResult
	content, _, err := s.chatSynthesis(cctx, prompt, func(body string) error {
		p, err := parseMarketSynthesisJSON(body)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}
		if strings.TrimSpace(p.Report) == "" {
			return fmt.Errorf("report empty")
		}
		if strings.TrimSpace(p.Summary) == "" {
			return fmt.Errorf("summary empty")
		}
		p.Result = normalizeMarketResult(p.Result)
		p.Confidence = normalizeMarketConfidence(p.Confidence)
		if p.Result == "" {
			return fmt.Errorf("result invalid")
		}
		if p.Confidence == "" {
			return fmt.Errorf("confidence invalid")
		}
		if len([]rune(p.Summary)) > 220 {
			return fmt.Errorf("summary too long (%d chars)", len([]rune(p.Summary)))
		}
		parsed = p
		return nil
	})
	if err != nil {
		return MarketSynthesisResult{}, fmt.Errorf("market synthesis: %w", err)
	}
	_ = content
	return parsed, nil
}

func buildMarketSynthesisPrompt(
	market, draft string,
	mc memory.MarketContext,
	evidence []memory.EvidenceRef,
	template string,
) []llm.Message {
	var b strings.Builder
	b.WriteString("你是市场盘前报告综合器。只能引用下面提供的证据与草稿，禁止编造价格、指数点位、新闻或任何未给出的数据。\n\n")
	b.WriteString(fmt.Sprintf("市场: %s\n\n", strings.ToUpper(strings.TrimSpace(market))))
	if strings.TrimSpace(template) != "" {
		b.WriteString("报告模板（仅使用与当前市场相关的章节，忽略其它市场注释块）:\n")
		b.WriteString(template)
		b.WriteString("\n\n")
	}
	b.WriteString("规则草稿（可改写为更通顺的中文报告，但不得添加新事实）:\n")
	b.WriteString(nonEmpty(draft, "(无草稿)"))
	b.WriteString("\n\n指数 hourly 分析:\n")
	for _, idx := range marketIndexOrder(market) {
		if summary, ok := mc.IndexAnalysisRefs[idx.code]; ok {
			b.WriteString(fmt.Sprintf("- %s (%s): %s\n", idx.name, idx.code, summary))
		}
	}
	for code, summary := range mc.IndexAnalysisRefs {
		if marketIndexKnown(market, code) {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", code, summary))
	}
	if len(mc.IndexAnalysisRefs) == 0 {
		b.WriteString("- (无指数分析)\n")
	}
	b.WriteString("\n市场新闻:\n")
	if news := strings.TrimSpace(mc.MarketNews[strings.ToUpper(strings.TrimSpace(market))]); news != "" {
		b.WriteString(news)
	} else {
		b.WriteString("(无新闻)")
	}
	b.WriteString("\n\n证据 refs:\n")
	for _, ev := range evidence {
		b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", ev.ID, ev.Source, ev.Summary))
	}
	if len(evidence) == 0 {
		b.WriteString("- (无 evidence ref，仅使用上文数据)\n")
	}
	b.WriteString(`
要求:
- report: 完整 Markdown，遵循模板结构，仅覆盖当前市场
- result: long / short / neutral 之一（市场短期情绪）
- confidence: high / medium / low 之一
- summary: <=200字，面向用户的一句话结论
- 禁止输出其它市场（CN/HK/US 中非当前市场）的指数或新闻
- 数据缺口写「暂无」，不要猜
- 禁止使用 Markdown 表格（含 | 列 | 值 | 形式）
- 禁止在 report 正文中写标题行（#）、生成时间、市场代码、report_date 等元数据
- 章节必须以二级标题开头，且标题文字固定为：## 指数概览、## 市场新闻解读、## 市场综合判断（不可省略 ##）
- 新闻解读：提取 3–5 条要点，禁止罗列发布时间或 🕐 时间戳；必须包含「**新闻面判断**」行（偏多/偏空/中性 + 理由）
- 市场综合判断：仅写综合结论段落及主要风险/今日关注，**禁止在 report 正文中写市场情绪、置信度、result、confidence**
- 正文末尾仅一行脚注：*报告由 GeeGoo 智能体市场盘前 skill 生成*（不要写 geegoo-agent 版本号或下次更新时间）
- 输出严格 JSON: {"report":"...","result":"...","confidence":"...","summary":"..."}`)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: "你是严格的 JSON 市场报告综合器, 只输出 JSON, 不输出任何其它文字。"},
		{Role: llm.RoleUser, Content: b.String()},
	}
}

type marketIndexSpec struct {
	name, code string
}

func marketIndexOrder(market string) []marketIndexSpec {
	switch strings.ToUpper(strings.TrimSpace(market)) {
	case "CN":
		return []marketIndexSpec{{"上证指数", "000001.SH"}, {"深证成指", "399001.SZ"}}
	case "HK":
		return []marketIndexSpec{{"恒生指数", "800000.HK"}}
	case "US":
		return []marketIndexSpec{{"道琼斯", "^DJI.US"}, {"纳斯达克", "^IXIC.US"}}
	default:
		return nil
	}
}

func marketIndexKnown(market, code string) bool {
	for _, idx := range marketIndexOrder(market) {
		if idx.code == code {
			return true
		}
	}
	return false
}

func parseMarketSynthesisJSON(content string) (MarketSynthesisResult, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var out MarketSynthesisResult
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

func normalizeMarketResult(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "long", "bullish", "positive":
		return "long"
	case "short", "bearish", "negative":
		return "short"
	case "neutral", "mixed", "flat":
		return "neutral"
	default:
		return ""
	}
}

func normalizeMarketConfidence(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "high":
		return "high"
	case "medium", "med":
		return "medium"
	case "low":
		return "low"
	default:
		return ""
	}
}
