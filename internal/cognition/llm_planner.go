package cognition

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const classifyTimeout = 8 * time.Second

const classifyPrompt = `You classify one user chat turn for a finance assistant.
Reply with ONLY JSON:
{"domain":"<one>","confidence":0.0,"reason":"<short>"}

Allowed domain values:
chat, stock_analysis, news, knowledge, report_lookup, report_write, bot_manage,
signal_probe, backtest_run, backtest_history, custom_signal, prompt_admin, dca_grid, ambiguous

Hard rules:
- backtest_run ONLY if the user explicitly asked to run a backtest (回测 / 跑回测 / 再回测 / backtest).
- Mentioning MACD, RSI, 信号, or 策略 without an explicit backtest verb is stock_analysis or ambiguous, never backtest_run.
- signal_probe only for 买卖点 / 测信号 / 有没有买卖.
- If unsure, use ambiguous.

Rule planner hint: domain=%s reason=%s
Last turn domain: %s
User: %s`

// IntentPlanner wraps a rule planner and optionally asks a small model
// to classify gray-zone turns. Nil LLM keeps pure rules.
type IntentPlanner struct {
	Rules Planner
	LLM   llm.Provider
}

// Plan implements Planner.
func (p IntentPlanner) Plan(in PlanInput) TurnPlan {
	rules := p.Rules
	if rules == nil {
		rules = RulePlanner{}
	}
	base := rules.Plan(in)
	if p.LLM == nil || !needsLLMAssist(base) {
		return base
	}
	got, ok := classifyWithLLM(in, p.LLM, base)
	if !ok {
		return base
	}
	return sanitizeLLMPlan(in, base, got)
}

func needsLLMAssist(base TurnPlan) bool {
	if base.Domain == DomainAmbiguous {
		return true
	}
	return base.Domain == DomainChat && base.Confidence < 0.75
}

type llmClassifyJSON struct {
	Domain     string  `json:"domain"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

func classifyWithLLM(in PlanInput, provider llm.Provider, base TurnPlan) (TurnPlan, bool) {
	ctx := in.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, classifyTimeout)
	defer cancel()

	prompt := fmt.Sprintf(classifyPrompt, base.Domain, base.Reason, in.LastDomain, strings.TrimSpace(in.UserText))
	resp, err := provider.Chat(ctx, []llm.Message{{
		Role:    llm.RoleUser,
		Content: prompt,
	}}, nil, 0.1, 200)
	if err != nil || resp == nil {
		return TurnPlan{}, false
	}
	text := strings.TrimSpace(resp.Content)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return TurnPlan{}, false
	}
	var parsed llmClassifyJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return TurnPlan{}, false
	}
	d := Domain(strings.TrimSpace(parsed.Domain))
	if !validDomain(d) {
		return TurnPlan{}, false
	}
	plan := planForDomain(d)
	if parsed.Reason != "" {
		plan.Reason = "llm: " + strings.TrimSpace(parsed.Reason)
	}
	if parsed.Confidence > 0 {
		plan.Confidence = parsed.Confidence
	}
	return plan, true
}

func sanitizeLLMPlan(in PlanInput, base, llmPlan TurnPlan) TurnPlan {
	if llmPlan.Domain == DomainBacktestRun && !isBacktestRun(in.UserText) {
		if base.Domain == DomainAmbiguous {
			return base
		}
		out := planForDomain(DomainChat)
		out.Reason = "拒绝无回测动词的 backtest_run"
		out.Confidence = 0.6
		return out
	}
	if llmPlan.Domain == DomainDCAGrid && !hasAny(in.UserText, dcaGridTokens) && base.Domain != DomainDCAGrid {
		return base
	}
	return llmPlan
}

func validDomain(d Domain) bool {
	switch d {
	case DomainChat, DomainStockAnalysis, DomainNews, DomainKnowledge,
		DomainReportLookup, DomainReportWrite, DomainBotManage,
		DomainSignalProbe, DomainBacktestRun, DomainBacktestHistory,
		DomainCustomSignal, DomainPromptAdmin, DomainDCAGrid, DomainAmbiguous:
		return true
	default:
		return false
	}
}
