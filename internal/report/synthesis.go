// Package report produces pre-market report content by combining rule-based
// structure with optional LLM synthesis. The LLM only synthesizes evidence
// already captured in working memory — it never invents prices, attitudes,
// or signals. When no LLM is configured or synthesis fails, the rule-based
// fallback in workflow.BuildReportContent is used unchanged.
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
// Result and Confidence are intentionally NOT here — those stay rule-based
// (attitude → result, evidence count → confidence) so the LLM cannot flip
// a neutral decision to long on its own.
type SynthesisResult struct {
	Reason     string `json:"reason"`
	Suggestion string `json:"suggestion"`
	Summary    string `json:"summary"`
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
		gateway: gateway, model: model, timeout: 60 * time.Second, maxTokens: 1024,
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
// the supplied evidence. On any error or contract violation, returns an error
// so the caller falls back to the rule-based path.
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
	callCtx := llm.WithCallMeta(cctx, llm.CallMeta{Kind: llm.TaskSynthesis})
	resp, err := s.gateway.Chat(callCtx, prompt, nil, "", 0)
	if err != nil {
		return SynthesisResult{}, fmt.Errorf("synthesis LLM call: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return SynthesisResult{}, fmt.Errorf("synthesis returned empty content")
	}
	parsed, err := parseSynthesisJSON(content)
	if err != nil {
		return SynthesisResult{}, fmt.Errorf("synthesis parse: %w", err)
	}
	if strings.TrimSpace(parsed.Reason) == "" {
		return SynthesisResult{}, fmt.Errorf("synthesis reason empty")
	}
	// Enforce minimum substance per rules/report-format.md (reason ≥ 80 chars).
	if len(parsed.Reason) < 80 {
		return SynthesisResult{}, fmt.Errorf("synthesis reason too short (%d chars)", len(parsed.Reason))
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
- reason: >=80字, 必须引用具体证据 ID 和数值, 禁止空洞表述如"综合来看偏乐观"
- suggestion: buy / sell / hold 之一
- summary: <=200字, 面向用户的一句话结论
- 只能使用上面给出的数据; 缺失的字段写"数据缺口", 不要猜
- 输出严格 JSON: {"reason":"...","suggestion":"...","summary":"..."}`)
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
