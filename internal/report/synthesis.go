// Package report produces pre-market report content by combining rule-based
// drafts with LLM synthesis. The LLM only synthesizes evidence already captured
// in working memory — it never invents prices, attitudes, or signals. Premarket
// market/stock report creation requires successful LLM synthesis; there is no
// rule-based fallback path for persisted reports.
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// SynthesisResult is the LLM-generated portion of a pre-market report.
// SuggestedResult/SuggestedConfidence are AI opinions; final result/confidence
// for stocks are decided by verdict.ArbitrateStockPreMarket (scheme B).
type SynthesisResult struct {
	SuggestedResult     string `json:"suggested_result"`
	SuggestedConfidence string `json:"suggested_confidence"`
	Reason              string `json:"reason"`
	Suggestion          string `json:"suggestion"`
	Summary             string `json:"summary"`
}

// Synthesizer calls an LLM gateway to synthesize report text from evidence.
type Synthesizer struct {
	gateway   *llm.Gateway
	model     string
	timeout   time.Duration
	maxTokens int
}

// NewSynthesizer creates a synthesizer. model is informational; the gateway's
// provider drives the actual model. timeout caps the LLM call.
func NewSynthesizer(gateway *llm.Gateway, model string) *Synthesizer {
	return &Synthesizer{
		// Primary (90s) + fallback (120s) synthesis budgets; outer cap prevents hung workflows.
		gateway: gateway, model: model, timeout: 240 * time.Second, maxTokens: 1024,
	}
}

// Available reports whether synthesis can run (gateway present).
func (s *Synthesizer) Available() bool { return s != nil && s.gateway != nil }

// SetGateway swaps the LLM gateway used for synthesis (e.g. after /model).
func (s *Synthesizer) SetGateway(gateway *llm.Gateway) {
	if s == nil {
		return
	}
	s.gateway = gateway
}

// Synthesize asks the LLM to write reason/suggestion/summary strictly from
// the supplied evidence. On any error or contract violation, returns an error.
func (s *Synthesizer) Synthesize(
	ctx context.Context,
	ws memory.StockWorkspace,
	evidence []memory.EvidenceRef,
	marketContext memory.MarketContext,
) (SynthesisResult, error) {
	if !s.Available() {
		return SynthesisResult{}, fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildSynthesisPrompt(ws, evidence, marketContext)
	var parsed SynthesisResult
	_, _, err := s.chatSynthesis(cctx, prompt, func(body string) error {
		p, err := parseSynthesisJSON(body)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}
		if strings.TrimSpace(p.Reason) == "" {
			return fmt.Errorf("reason empty")
		}
		p.SuggestedResult = normalizeMarketResult(p.SuggestedResult)
		p.SuggestedConfidence = normalizeMarketConfidence(p.SuggestedConfidence)
		if p.SuggestedResult == "" {
			p.SuggestedResult = "neutral"
		}
		if p.SuggestedConfidence == "" {
			p.SuggestedConfidence = "low"
		}
		if len(p.Reason) < 80 {
			return fmt.Errorf("reason too short (%d chars)", len(p.Reason))
		}
		parsed = p
		return nil
	})
	if err != nil {
		return SynthesisResult{}, fmt.Errorf("synthesis: %w", err)
	}
	return parsed, nil
}

func buildSynthesisPrompt(ws memory.StockWorkspace, evidence []memory.EvidenceRef, mc memory.MarketContext) []llm.Message {
	var b strings.Builder
	b.WriteString("你是盘前报告综合器。只能引用下面提供的证据，禁止编造价格、态度、资金流或任何未给出的数据。\n\n")
	b.WriteString(fmt.Sprintf("标的: %s (%s)\n", ws.StockName, ws.Code))
	b.WriteString(fmt.Sprintf("Bot 昨日态度: %s\n", nonEmpty(ws.Attitude, "neutral")))
	b.WriteString("\n已捕获证据 (evidence refs):\n")
	for _, ev := range evidence {
		b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", ev.ID, ev.Source, ev.Summary))
	}
	if len(evidence) == 0 {
		b.WriteString("- (无证据)\n")
	}
	b.WriteString("\n市场概况摘要:\n")
	for k, v := range mc.MarketNews {
		b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}
	for code, a := range mc.IndexAnalysisRefs {
		b.WriteString(fmt.Sprintf("- 指数 %s: %s\n", code, a))
	}
	b.WriteString("\n周线技术分析: " + nonEmpty(ws.WeeklyAnalysisRef, "(未捕获)") + "\n")
	b.WriteString("资金流: " + nonEmpty(ws.CapitalFlowSummary, "(未捕获)") + "\n")
	b.WriteString("资金分布: " + nonEmpty(ws.CapitalDistributionSummary, "(未捕获)") + "\n")
	b.WriteString("个股新闻: " + nonEmpty(ws.StockNewsSummary, "(未捕获)") + "\n")
	b.WriteString(`
要求:
- suggested_result: long / short / neutral 之一（基于证据的市场方向建议，非最终裁决）
- suggested_confidence: high / medium / low 之一
- reason: >=80字, 必须引用具体证据 ID 和数值, 禁止空洞表述如"综合来看偏乐观"
- suggestion: buy / sell / hold 之一
- summary: <=200字, 面向用户的一句话结论
- 只能使用上面给出的数据; 缺失的字段写"数据缺口", 不要猜
- 输出严格 JSON: {"suggested_result":"...","suggested_confidence":"...","reason":"...","suggestion":"...","summary":"..."}`)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: "你是严格的 JSON 报告综合器, 只输出 JSON, 不输出任何其它文字。"},
		{Role: llm.RoleUser, Content: b.String()},
	}
}

func parseSynthesisJSON(content string) (SynthesisResult, error) {
	content = strings.TrimSpace(content)
	// Strip markdown code fences if present.
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var out SynthesisResult
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		// Tolerate trailing text after JSON by extracting the object span.
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

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
