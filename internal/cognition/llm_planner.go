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

const classifyTimeout = 12 * time.Second
const classifyMaxAttempts = 3
const symbolCountTimeout = 4 * time.Second

const symbolCountPrompt = `Count how many distinct stocks/companies the user wants quoted or analyzed in this message.
Reply ONLY JSON: {"symbol_count":0,"reason":"short"}

Rules:
- Each separate company the user wants addressed counts as 1 (e.g. 腾讯 + 阿里巴巴 = 2).
- Multiple listings of the same company (港股/美股) count as 1 unless user asks to compare listings.
- If no stock/company is mentioned, symbol_count=0.

User: %s`

const classifyPromptCore = `You classify one user chat turn for a finance assistant.
Reply with ONLY JSON:
{"domain":"<one>","mode":"<one>","act":"<optional>","symbol_count":0,"clarify":"<optional>","confidence":0.0,"reason":"<short>"}

Allowed domain values:
chat, stock_analysis, news, knowledge, report_lookup, report_write, bot_manage,
signal_probe, backtest_run, backtest_history, custom_signal, prompt_admin, dca_grid, ambiguous

Allowed mode values:
talk, gather, execute, clarify

When domain is stock_analysis, act MUST be one of:
analyze, quote_price, technical_analysis, context_followup, symbol_resolve, multi_symbol_delegate
- quote_price: user wants a price snapshot / quote only for exactly ONE symbol (查询/查一下/现价/多少钱/股价是多少).
  Example: "帮我查询下腾讯股价" → quote_price
  Do NOT use quote_price when 2+ companies/symbols are named — use multi_symbol_delegate instead.
- technical_analysis: user wants analysis of trend, K-line, technicals, or price movement over a period for ONE symbol.
  Example: "帮我分析下腾讯最近一个月的价格走势" → technical_analysis
- multi_symbol_delegate: user asks to analyze, quote, or compare TWO OR MORE distinct stocks/companies in the same turn. Label only — main ReAct may delegate via delegate_tasks (preferred) or call stock tools directly; do not treat this as a hard tool restriction.
  Example: "请帮我分析下腾讯和阿里巴巴最近的股价" → multi_symbol_delegate
  Do NOT use for a single symbol even if the wording includes "分析" and "股价".
- analyze: general stock analysis for ONE symbol when none of the above fits
  Example: "帮我分析一下中际旭创" → stock_analysis/gather act=analyze
- context_followup: pronoun or short follow-up continuing the same symbol in session
  Example (last turn stock_analysis): "它最近走势怎么样" → stock_analysis/gather act=context_followup
- symbol_resolve: user explicitly switches to a different stock symbol
When domain is stock_analysis, set symbol_count to the number of distinct stocks/companies the user wants analyzed or quoted in this turn (0 if none named).
If symbol_count >= 2 and mode is gather, act MUST be multi_symbol_delegate (overrides technical_analysis/analyze).
If unsure whether the user wants a price snapshot (quote_price) or price/trend analysis (technical_analysis),
use domain=ambiguous, mode=clarify, clarify=stock_quote (do NOT guess).
For non-stock_analysis domains, omit act or use empty string.

Optional clarify field (only when domain=ambiguous and mode=clarify):
- stock_quote: user mentioned a stock price but quote vs analysis is unclear (e.g. "腾讯股价怎么样")
- signal_usage: user asks how to use a MACD/signal without naming the catalog entry (e.g. "这个MACD信号平时该怎么用")
- compound_steps: one sentence asks for both stock analysis and backtest (e.g. "分析一下再跑回测")

Domain + mode guidance:
- stock_analysis/gather: analyze a stock, quote, technicals, trends.
- signal_probe/execute: probe buy/sell points (买卖点 / 测信号 / 有没有买卖), not full PnL backtest.
- backtest_run/execute: explicit backtest request (回测 / 跑回测 / backtest). Tool chain uses run_strategy_backtest (catalog signal backtest), NOT loopback_strategy, unless user explicitly asks DCA/网格/定投 loopback.
- dca_grid/gather: list or browse available signal strategies/combinations (有哪些信号/策略/组合).
- dca_grid/execute: DCA or grid strategy plan/generation; loopback_strategy only after generate_* when validating a DCA/Grid bot plan.
- bot_manage/gather: list/query bots, reminders, SmartTrade, grid PnL.
- bot_manage/execute: create/update/delete bots.
- chat/talk: definitions, chitchat, signal quality opinions (准吗/靠谱吗) after prior context.
  Use chat/talk for general concept/indicator definition questions even if a knowledge base exists:
  e.g. "MACD 指标是什么意思" / "什么是 RSI" / "MACD是什么" → chat/talk (NOT knowledge).
  Do NOT use chat/talk when the user names a stock/company or asks for quote, analysis, or trend.
- knowledge/gather: ONLY when the user explicitly asks to use/search the knowledge base or internal docs
  (e.g. "按知识库", "知识库里", "查知识库", "根据文档").
  Do NOT route "X是什么意思" to knowledge just because X is a trading term.
- ambiguous/clarify: bare strategy words (MACD/SAR) without clear action, or compound analyze+backtest in one sentence.
  "这个MACD信号平时该怎么用" → ambiguous/clarify signal_usage (NOT chat/talk, NOT knowledge).
- Read Session summary and Recent session dialogue like a coding assistant: short follow-ups continue the active task.
- Use last turn domain + session dialogue for follow-ups; do not re-classify the task from scratch.
- If recent dialogue shows signal_probe/backtest_run/stock_analysis work and this turn only swaps strategy,
  symbol, or parameters (换策略/换一个/再来一次/再看看), keep the same domain/mode as the active task.
- If last turn domain is signal_probe or backtest_run and this turn does not clearly switch task
  (new analysis/quote, list strategies, quality opinion, or an explicit backtest verb from a non-backtest turn),
  keep that same domain and execute. Missing 测信号/回测 in a follow-up is NOT grounds for ambiguous.
- When session dialogue shows an active task, do NOT return ambiguous/clarify with a generic intent question;
  only clarify missing slots (strategy name, symbol) inside execute/gather, or use domain-specific clarify kinds.

Hard rules:
- backtest_run for a NEW task needs an explicit backtest verb; continuing last-turn backtest_run does not.
- backtest_run/execute: primary tool is run_strategy_backtest; do not route to loopback_strategy or generate_dca_strategy unless user explicitly wants DCA/Grid bot backtest.
- signal_probe ONLY for buy/sell point probing, not strategy listing.
- stock_analysis with 2+ distinct symbols/companies to address in one turn → act MUST be multi_symbol_delegate (not technical_analysis or analyze).
- If unsure, use ambiguous/clarify.`

// IntentPlanner is the sole production planner: LLM classify with structured
// clarify-choice shortcuts and fail-fast when classification is unavailable.
type IntentPlanner struct {
	LLM         llm.Provider
	LLMResolver func() llm.Provider
}

func (p IntentPlanner) resolveLLM() llm.Provider {
	if p.LLMResolver != nil {
		if provider := p.LLMResolver(); provider != nil {
			return provider
		}
	}
	return p.LLM
}

// Plan implements Planner.
func (p IntentPlanner) Plan(in PlanInput) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	if in.LastDomain == DomainAmbiguous {
		if d, act, ok := mapClarifyChoice(msg); ok {
			plan := planForDomain(d)
			if act != "" {
				plan.Act = act
			}
			plan.Reason = "用户选择了上一轮澄清选项"
			plan.Confidence = 0.9
			return applyPlanToolPolicies(plan)
		}
	}

	if plan, ok := deterministicPreLLMPlan(in); ok {
		return applyPlanToolPolicies(plan)
	}

	provider := p.resolveLLM()
	if provider == nil {
		return classifyFailedPlan("llm provider not configured")
	}
	plan, errDetail := classifyWithLLM(in, provider)
	if errDetail != "" {
		return classifyFailedPlan(errDetail)
	}
	return applyPlanToolPolicies(applyActiveTaskGuard(in, applyStickySessionPlan(in, sanitizeLLMPlan(in, plan))))
}

type llmClassifyJSON struct {
	Domain      string  `json:"domain"`
	Mode        string  `json:"mode"`
	Act         string  `json:"act"`
	SymbolCount int     `json:"symbol_count"`
	Clarify     string  `json:"clarify"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason"`
}

func classifyWithLLM(in PlanInput, provider llm.Provider) (TurnPlan, string) {
	var lastErr string
	for attempt := 0; attempt < classifyMaxAttempts; attempt++ {
		plan, errDetail := classifyOnce(in, provider, attempt > 0)
		if errDetail != "" {
			lastErr = errDetail
			continue
		}
		msg := strings.TrimSpace(in.UserText)
		if plan.Domain == DomainChat && plan.Mode == ModeTalk && len([]rune(msg)) >= 8 {
			retry, errDetail2 := classifyOnce(in, provider, true)
			if errDetail2 == "" && retry.Domain != DomainChat && retry.Domain != DomainAmbiguous {
				return retry, ""
			}
		}
		return plan, ""
	}
	if lastErr == "" {
		lastErr = "llm: classify failed"
	}
	return TurnPlan{}, lastErr
}

func classifyOnce(in PlanInput, provider llm.Provider, retry bool) (TurnPlan, string) {
	base := context.Background()
	if in.Ctx != nil {
		base = context.WithoutCancel(in.Ctx)
	}
	ctx, cancel := context.WithTimeout(base, classifyTimeout)
	defer cancel()

	prompt := buildClassifyPrompt(in)
	if retry {
		prompt += "\n\nRetry: if the user mentions a stock/company or asks for quote, analysis, trend, or continues prior session context, do NOT return chat/talk or generic ambiguous/clarify."
	}
	resp, err := provider.Chat(ctx, []llm.Message{{
		Role:    llm.RoleUser,
		Content: prompt,
	}}, nil, 0.1, 512)
	if err != nil {
		return TurnPlan{}, fmt.Sprintf("llm: %v", err)
	}
	if resp == nil {
		return TurnPlan{}, "llm: empty response"
	}
	text := strings.TrimSpace(resp.Content)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return TurnPlan{}, "llm: invalid json"
	}
	var parsed llmClassifyJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return TurnPlan{}, fmt.Sprintf("llm: json parse: %v", err)
	}
	d := Domain(strings.TrimSpace(parsed.Domain))
	if !validDomain(d) {
		return TurnPlan{}, fmt.Sprintf("llm: invalid domain %q", parsed.Domain)
	}
	plan := planForDomain(d)
	if m := Mode(strings.TrimSpace(parsed.Mode)); validMode(m) {
		plan.Mode = m
		if m == ModeClarify || d == DomainAmbiguous {
			kind := strings.TrimSpace(parsed.Clarify)
			if kind == "" {
				kind = inferAmbiguousClarifyKind(strings.TrimSpace(in.UserText))
			}
			plan = applyClarifyTemplate(plan, kind)
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
		plan = applyStockSymbolCountAct(plan, parsed.SymbolCount)
		if plan.Mode == ModeGather && parsed.SymbolCount < 2 && shouldRefineSymbolCount(plan.Act, in.UserText) {
			if count, ok := countStockSymbolsWithLLM(in, provider); ok && count >= 2 {
				plan = applyStockSymbolCountAct(plan, count)
			}
		}
	}
	return plan, ""
}

type symbolCountJSON struct {
	SymbolCount int    `json:"symbol_count"`
	Reason      string `json:"reason"`
}

func countStockSymbolsWithLLM(in PlanInput, provider llm.Provider) (int, bool) {
	if provider == nil {
		return 0, false
	}
	base := context.Background()
	if in.Ctx != nil {
		base = context.WithoutCancel(in.Ctx)
	}
	ctx, cancel := context.WithTimeout(base, symbolCountTimeout)
	defer cancel()

	userText := strings.TrimSpace(in.UserText)
	if userText == "" {
		return 0, false
	}
	resp, err := provider.Chat(ctx, []llm.Message{{
		Role:    llm.RoleUser,
		Content: fmt.Sprintf(symbolCountPrompt, userText),
	}}, nil, 0.1, 80)
	if err != nil || resp == nil {
		return 0, false
	}
	text := strings.TrimSpace(resp.Content)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return 0, false
	}
	var parsed symbolCountJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		return 0, false
	}
	if parsed.SymbolCount < 0 {
		return 0, false
	}
	return parsed.SymbolCount, true
}

func shouldRefineSymbolCount(act, userText string) bool {
	switch domaincatalog.NormalizeStockAct(act) {
	case domaincatalog.StockActQuotePrice, domaincatalog.StockActTechnicalAnalysis:
		return true
	case domaincatalog.StockActAnalyze:
		return len([]rune(strings.TrimSpace(userText))) >= 12
	default:
		return false
	}
}

// applyStockSymbolCountAct promotes gather turns with 2+ symbols to multi_symbol_delegate
// using the classifier's symbol_count (LLM-estimated, not keyword rules).
func applyStockSymbolCountAct(plan TurnPlan, symbolCount int) TurnPlan {
	if plan.Domain != DomainStockAnalysis || plan.Mode != ModeGather || symbolCount < 2 {
		return plan
	}
	switch domaincatalog.NormalizeStockAct(plan.Act) {
	case domaincatalog.StockActContextFollowup, domaincatalog.StockActSymbolResolve, domaincatalog.StockActMultiSymbol:
		return plan
	default:
		plan.Act = domaincatalog.StockActMultiSymbol
		plan.Reason = fmt.Sprintf("llm: symbol_count=%d → multi_symbol_delegate (%s)", symbolCount, plan.Reason)
		return plan
	}
}

func applyActiveTaskGuard(in PlanInput, plan TurnPlan) TurnPlan {
	if !isActiveTaskDomain(in.LastDomain) {
		return plan
	}
	if plan.Domain == DomainAmbiguous && plan.Mode == ModeClarify {
		return inheritLastTask(in, plan, "active task: skip generic intent clarify")
	}
	return plan
}

func applyStickySessionPlan(in PlanInput, plan TurnPlan) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	if isBacktestRun(msg) {
		return plan
	}
	if !isStickyDomain(in.LastDomain) {
		return plan
	}
	// Last turn already chose a task. Ambiguous = unsure this utterance, not a new session.
	if plan.Domain == DomainAmbiguous && plan.Mode == ModeClarify && !isQualityOpinion(msg) {
		if isActiveTaskDomain(in.LastDomain) {
			return inheritLastTask(in, plan, "sticky session: keep last task instead of re-asking intent")
		}
		if in.HasSessionHistory() && isStickyDomain(in.LastDomain) && !isExplicitTaskSwitch(plan, msg) {
			return inheritLastTask(in, plan, "session context: continue prior task instead of generic clarify")
		}
	}
	if plan.Domain != DomainChat || plan.Mode != ModeTalk {
		return plan
	}
	if isQualityOpinion(msg) {
		return plan
	}
	if !isFollowUpUtterance(msg) && len([]rune(msg)) > 24 {
		return plan
	}
	return inheritLastTask(in, plan, "sticky session: short follow-up overrides chat misroute")
}

func inheritLastTask(in PlanInput, plan TurnPlan, reason string) TurnPlan {
	out := planForDomain(in.LastDomain)
	if in.LastDomain == DomainStockAnalysis {
		out.Act = domaincatalog.StockActContextFollowup
	}
	out.Reason = reason
	if plan.Confidence > 0 {
		out.Confidence = plan.Confidence
	}
	return out
}

func sanitizeLLMPlan(in PlanInput, llmPlan TurnPlan) TurnPlan {
	msg := strings.TrimSpace(in.UserText)
	if isBacktestRun(msg) {
		out := planForDomain(DomainBacktestRun)
		out.Mode = ModeExecute
		out.Reason = "显式回测动词 → backtest_run"
		if llmPlan.Confidence > 0 {
			out.Confidence = llmPlan.Confidence
		}
		return out
	}
	if llmPlan.Domain == DomainBacktestRun && !isBacktestRun(in.UserText) {
		if in.LastDomain == DomainBacktestRun {
			return llmPlan
		}
		if in.LastDomain == DomainAmbiguous {
			p := planForDomain(DomainAmbiguous)
			p.Reason = "拒绝无回测动词的 backtest_run"
			return p
		}
		out := planForDomain(DomainChat)
		out.Reason = "拒绝无回测动词的 backtest_run"
		out.Confidence = 0.6
		return out
	}
	if llmPlan.Domain == DomainDCAGrid && llmPlan.Mode == ModeExecute &&
		!hasAny(in.UserText, dcaGridTokens) {
		out := planForDomain(DomainChat)
		out.Reason = "拒绝无 DCA/网格上下文的 dca_grid execute"
		out.Confidence = 0.6
		return out
	}
	return llmPlan
}

func applyPlanToolPolicies(plan TurnPlan) TurnPlan {
	return applyBacktestRunToolPolicy(applyGatherToolPolicy(plan))
}

func applyGatherToolPolicy(plan TurnPlan) TurnPlan {
	if plan.Domain == DomainDCAGrid && plan.Mode == ModeGather {
		plan.ToolsAllow = filterTools(plan.ToolsAllow, "run_strategy_backtest")
	}
	return plan
}

func applyBacktestRunToolPolicy(plan TurnPlan) TurnPlan {
	if plan.Domain == DomainBacktestRun && plan.Mode == ModeExecute {
		plan.ToolsAllow = filterTools(plan.ToolsAllow,
			"loopback_strategy", "generate_dca_strategy", "generate_grid_strategy")
	}
	return plan
}

func filterTools(in []string, drop ...string) []string {
	if len(in) == 0 || len(drop) == 0 {
		return in
	}
	banned := map[string]struct{}{}
	for _, name := range drop {
		banned[name] = struct{}{}
	}
	out := make([]string, 0, len(in))
	for _, name := range in {
		if _, ok := banned[name]; ok {
			continue
		}
		out = append(out, name)
	}
	return out
}

func deterministicPreLLMPlan(in PlanInput) (TurnPlan, bool) {
	msg := strings.TrimSpace(in.UserText)
	if isBacktestRun(msg) {
		p := planForDomain(DomainBacktestRun)
		p.Mode = ModeExecute
		p.Reason = "显式回测动词"
		p.Confidence = 0.85
		return p, true
	}
	return TurnPlan{}, false
}

func classifyFailedPlan(detail string) TurnPlan {
	return TurnPlan{
		Reason:        "classify_failed",
		ClassifyError: strings.TrimSpace(detail),
	}
}

func applyClarifyTemplate(plan TurnPlan, template string) TurnPlan {
	switch template {
	case "stock_quote":
		plan.ClarifyQuestion = domaincatalog.StockPriceClarifyQuestion
		plan.ClarifyChoices = append([]string(nil), domaincatalog.StockPriceClarifyChoices...)
		plan.ToolsAllow = []string{"clarify"}
		return plan
	case "signal_usage":
		plan.ClarifyQuestion = domaincatalog.SignalUsageClarifyQuestion
		plan.ClarifyChoices = append([]string(nil), domaincatalog.SignalUsageClarifyChoices...)
		plan.ToolsAllow = []string{"clarify"}
		return plan
	case "compound_steps":
		plan.ClarifyQuestion = domaincatalog.CompoundStepClarifyQuestion
		plan.ClarifyChoices = append([]string(nil), domaincatalog.CompoundStepClarifyChoices...)
		plan.ToolsAllow = []string{"clarify"}
		return plan
	default:
		return applyAmbiguousClarify(plan)
	}
}

func inferAmbiguousClarifyKind(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	if hasAny(msg, []string{"股价", "现价", "多少钱", "报价"}) &&
		hasAny(msg, []string{"怎么样", "如何", "咋样", "怎样"}) {
		return "stock_quote"
	}
	if hasAny(msg, []string{"分析", "看看", "解读"}) && hasAny(msg, []string{"回测", "跑回测"}) {
		return "compound_steps"
	}
	if hasAny(msg, []string{"MACD", "SAR", "RSI", "EMA", "信号"}) &&
		hasAny(msg, []string{"怎么用", "用法", "比较好", "如何用", "怎么弄", "日常"}) {
		return "signal_usage"
	}
	return ""
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
