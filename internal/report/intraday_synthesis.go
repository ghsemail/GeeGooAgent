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

// IntradaySynthesisResult is the LLM intraday decision bundle.
type IntradaySynthesisResult struct {
	Result     string `json:"result"`
	Confidence string `json:"confidence"`
	Summary    string `json:"summary"`
	Reason     string `json:"reason"`
}

// SynthesizeIntraday decides buy/sell/hold from collected intraday evidence.
func (s *Synthesizer) SynthesizeIntraday(
	ctx context.Context,
	ws memory.StockWorkspace,
	draft string,
	ruleResult, ruleConfidence string,
) (IntradaySynthesisResult, error) {
	if !s.Available() {
		return IntradaySynthesisResult{}, fmt.Errorf("synthesizer not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildIntradaySynthesisPrompt(ws, draft, ruleResult, ruleConfidence)
	content, _, err := s.chatSynthesis(cctx, prompt, func(body string) error {
		_, err := parseIntradaySynthesisJSON(body)
		return err
	})
	if err != nil {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis LLM call: %w", err)
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis returned empty content")
	}
	parsed, err := parseIntradaySynthesisJSON(content)
	if err != nil {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis parse: %w", err)
	}
	parsed.Result = normalizeIntradayResult(parsed.Result)
	parsed.Confidence = normalizeMarketConfidence(parsed.Confidence)
	if parsed.Result == "" {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis result invalid")
	}
	if parsed.Confidence == "" {
		return IntradaySynthesisResult{}, fmt.Errorf("intraday synthesis confidence invalid")
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

func buildIntradaySynthesisPrompt(ws memory.StockWorkspace, draft, ruleResult, ruleConfidence string) []llm.Message {
	var b strings.Builder
	b.WriteString("你是个股盘中交易决策器。只能引用下列事实与报告草稿，禁止编造未给出的价格、持仓或指标。\n\n")
	b.WriteString(fmt.Sprintf("标的: %s (%s)\n", ws.StockName, ws.Code))
	b.WriteString(fmt.Sprintf("触发 Bot: %s（%s，类型 %s）\n", ws.BotName, ws.BotID, ws.BotType))
	b.WriteString(fmt.Sprintf("本轮信号: %s\n", ws.TradeType))
	if ws.AttitudeSwitch {
		if strings.TrimSpace(ws.PreMarketResult) != "" {
			b.WriteString(fmt.Sprintf("盘前: %s（置信 %s）\n", ws.PreMarketResult, ws.PreMarketConfidence))
			if strings.TrimSpace(ws.PreMarketReason) != "" {
				b.WriteString("盘前依据: " + truncateLine(ws.PreMarketReason, 400) + "\n")
			}
		} else {
			b.WriteString("盘前: 已启用但暂无报告，不作为方向依据\n")
		}
	} else {
		b.WriteString("盘前: 未启用（attitude.switch 关闭，不读取盘前报告）\n")
	}
	if !isReminderBotType(ws.BotType) {
		if strings.TrimSpace(ws.PositionSummary) != "" {
			b.WriteString("持仓: " + strings.TrimSpace(ws.PositionSummary) + "\n")
		} else {
			b.WriteString("持仓: 无持仓或未获取\n")
		}
	}
	if strings.TrimSpace(ws.CapitalDistributionSummary) != "" {
		b.WriteString("资金分布: " + truncateLine(ws.CapitalDistributionSummary, 500) + "\n")
	}
	if ws.CurrentPrice > 0 {
		b.WriteString(fmt.Sprintf("参考价: %s（来源 %s）\n", stockfmt.FormatPrice(ws.CurrentPrice), ws.PriceSource))
	}
	b.WriteString(fmt.Sprintf("\n规则引擎参考（可采纳或否决，但须说明理由）: %s / %s\n\n", ruleResult, ruleConfidence))
	b.WriteString("已排版报告草稿:\n")
	b.WriteString(nonEmpty(draft, "(无草稿)"))
	b.WriteString(`

要求:
- 综合盘前、持仓、资金分布、小时级分析与参考价，对「本轮信号」给出是否执行的最终决策
- result: 仅 buy | sell | hold（分别表示批准买入、批准卖出、观望不执行）
- confidence: high | medium | low
- summary: <=200字中文结论；禁止 Markdown、标签、英文枚举
- reason: >=80字中文，说明为何批准或否决本轮信号；禁止 Markdown 表格、[ev_...]、科学计数法
- 硬约束: 非 Reminder 的卖出信号在无持仓时必须 hold；盘前高置信看空时买入信号应倾向 hold
- 输出严格 JSON: {"result":"buy|sell|hold","confidence":"high|medium|low","summary":"...","reason":"..."}`)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: "你是严格的 JSON 盘中决策器, 只输出 JSON, 不输出任何其它文字。"},
		{Role: llm.RoleUser, Content: b.String()},
	}
}

func isReminderBotType(botType string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(botType)), "reminder")
}

func truncateLine(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func normalizeIntradayResult(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "buy", "long", "买入", "看多":
		return "buy"
	case "sell", "short", "卖出", "看空":
		return "sell"
	case "hold", "neutral", "观望", "中性":
		return "hold"
	default:
		return ""
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
