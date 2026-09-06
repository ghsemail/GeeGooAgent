package cognition

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

const classifyTimeout = 2 * time.Second

const classifyPrompt = `You classify one user chat turn for a finance assistant.
Reply with ONLY JSON:
{"domain":"<one>","mode":"<one>","confidence":0.0,"reason":"<short>"}

Allowed domain values:
chat, stock_analysis, news, knowledge, report_lookup, report_write, bot_manage,
signal_probe, backtest_run, backtest_history, custom_signal, prompt_admin, dca_grid, ambiguous

Allowed mode values:
talk, gather, execute, clarify

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
- Follow last turn domain for short follow-ups (它最近走势 / 不聊XX了 / 接着… / 刚才那次…) when the user did not switch topic.

Hard rules:
- backtest_run ONLY with an explicit backtest verb.
- signal_probe ONLY for buy/sell point probing, not strategy listing.
- If unsure, use ambiguous/clarify.

Rule hint (non-binding): domain=%s mode=%s reason=%s
Last turn domain: %s
User: %s`

// IntentPlanner classifies turns with an LLM when available. RulePlanner
// supplies a non-binding hint and serves as fallback when the LLM is nil or fails.
type IntentPlanner struct {
	Rules Planner
	LLM   llm.Provider
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

	rules := p.Rules
	if rules == nil {
		rules = RulePlanner{}
	}
	base := rules.Plan(in)
	if p.LLM == nil {
		return base
	}
	got, ok := classifyWithLLM(in, p.LLM, base)
	if !ok {
		return base
	}
	return sanitizeLLMPlan(in, base, got)
}

type llmClassifyJSON struct {
	Domain     string  `json:"domain"`
	Mode       string  `json:"mode"`
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

	prompt := fmt.Sprintf(classifyPrompt, base.Domain, base.Mode, base.Reason, in.LastDomain, strings.TrimSpace(in.UserText))
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
	if llmPlan.Domain == DomainDCAGrid && llmPlan.Mode == ModeExecute &&
		!hasAny(in.UserText, dcaGridTokens) && base.Domain != DomainDCAGrid {
		return base
	}
	return llmPlan
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
