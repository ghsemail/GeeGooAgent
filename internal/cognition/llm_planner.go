package cognition

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const classifyTimeout = 2 * time.Second

const classifyPrompt = `You classify one user chat turn for a finance assistant.
Reply with ONLY JSON:
{"domain":"<one>","mode":"<one>","act":"<optional>","confidence":0.0,"reason":"<short>"}

Allowed domain values:
chat, stock_analysis, news, knowledge, report_lookup, report_write, bot_manage,
signal_probe, backtest_run, backtest_history, custom_signal, prompt_admin, dca_grid, ambiguous

Allowed mode values:
talk, gather, execute, clarify

When domain is stock_analysis, act MUST be one of:
analyze, quote_price, technical_analysis, context_followup, symbol_resolve
- quote_price: user wants current price / quote only
- technical_analysis: technicals, K-line, indicators, trend analysis
- context_followup: pronoun or short follow-up continuing the same symbol in session
- symbol_resolve: user explicitly switches to a different stock symbol
- analyze: general stock analysis when none of the above fits
For non-stock_analysis domains, omit act or use empty string.

Domain + mode guidance:
- stock_analysis/gather: analyze a stock, quote, technicals, trends.
- signal_probe/execute: probe buy/sell points (买卖点 / 测信号 / 有没有买卖), not full PnL backtest.
- backtest_run/execute: explicit backtest request (回测 / 跑回测 / backtest).
- dca_grid/gather: list or browse available signal strategies/combinations (有哪些信号/策略/组合).
- dca_grid/execute: DCA or grid strategy backtest/generation.
- bot_manage/gather: list/query bots, reminders, SmartTrade, grid PnL.
- bot_manage/execute: create/update/delete bots.
- chat/talk: definitions, chitchat, signal quality opinions (准吗/靠谱吗) after prior context.
- ambiguous/clarify: bare strategy words (MACD/SAR) without clear action, or compound analyze+backtest in one sentence.
- Use last turn domain + dialogue context for short follow-ups; do not rely on single keywords alone.

Hard rules:
- backtest_run ONLY with an explicit backtest verb.
- signal_probe ONLY for buy/sell point probing, not strategy listing.
- If unsure, use ambiguous/clarify.

Last turn domain: %s
User: %s`

// IntentPlanner is the sole production planner: LLM classify with structured
// clarify-choice shortcuts and conservative fallback when the LLM is unavailable.
type IntentPlanner struct {
	LLM llm.Provider
}

// Plan implements Planner.
func (p IntentPlanner) Plan(in PlanInput) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	if in.LastDomain == DomainAmbiguous {
		if d, ok := mapClarifyChoice(msg); ok {
			plan := planForDomain(d)
			plan.Reason = "用户选择了上一轮澄清选项"
			plan.Confidence = 0.9
			return plan
		}
	}

	if p.LLM == nil {
		return plannerFallback(in)
	}
	got, ok := classifyWithLLM(in, p.LLM)
	if !ok {
		return plannerFallback(in)
	}
	return sanitizeLLMPlan(in, got)
}

type llmClassifyJSON struct {
	Domain     string  `json:"domain"`
	Mode       string  `json:"mode"`
	Act        string  `json:"act"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

func classifyWithLLM(in PlanInput, provider llm.Provider) (TurnPlan, bool) {
	ctx := in.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, classifyTimeout)
	defer cancel()

	prompt := fmt.Sprintf(classifyPrompt, in.LastDomain, strings.TrimSpace(in.UserText))
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
	if m := Mode(strings.TrimSpace(parsed.Mode)); validMode(m) {
		plan.Mode = m
		if m == ModeClarify || d == DomainAmbiguous {
			plan = applyAmbiguousClarify(plan)
		}
	}
	if parsed.Reason != "" {
		plan.Reason = "llm: " + strings.TrimSpace(parsed.Reason)
	}
	if parsed.Confidence > 0 {
		plan.Confidence = parsed.Confidence
	}
	if d == DomainStockAnalysis {
		plan.Act = domaincatalog.NormalizeStockAct(parsed.Act)
	}
	return plan, true
}

func sanitizeLLMPlan(in PlanInput, llmPlan TurnPlan) TurnPlan {
	fallback := plannerFallback(in)
	if llmPlan.Domain == DomainBacktestRun && !isBacktestRun(in.UserText) {
		if fallback.Domain == DomainAmbiguous {
			return fallback
		}
		out := planForDomain(DomainChat)
		out.Reason = "拒绝无回测动词的 backtest_run"
		out.Confidence = 0.6
		return out
	}
	if llmPlan.Domain == DomainDCAGrid && llmPlan.Mode == ModeExecute &&
		!hasAny(in.UserText, dcaGridTokens) && fallback.Domain != DomainDCAGrid {
		return fallback
	}
	return llmPlan
}

func plannerFallback(in PlanInput) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	if in.LastDomain == DomainAmbiguous {
		p := planForDomain(DomainAmbiguous)
		p.Reason = "fallback: 等待澄清"
		p.Confidence = 0.4
		return p
	}
	if isStickyDomain(in.LastDomain) && (isFollowUpUtterance(msg) || len([]rune(msg)) <= 24) {
		p := planForDomain(in.LastDomain)
		p.Reason = "fallback: 沿用上一轮领域 " + string(in.LastDomain)
		p.Confidence = 0.5
		return p
	}
	p := planForDomain(DomainChat)
	p.Reason = "fallback: LLM 不可用"
	p.Confidence = 0.3
	return p
}

func applyAmbiguousClarify(plan TurnPlan) TurnPlan {
	spec := planForDomain(DomainAmbiguous)
	plan.ClarifyQuestion = spec.ClarifyQuestion
	plan.ClarifyChoices = append([]string(nil), spec.ClarifyChoices...)
	plan.ToolsAllow = append([]string(nil), spec.ToolsAllow...)
	return plan
}

func validMode(m Mode) bool {
	switch m {
	case ModeTalk, ModeGather, ModeExecute, ModeClarify:
		return true
	default:
		return false
	}
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
