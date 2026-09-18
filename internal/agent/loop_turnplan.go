package agent

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func turnPlanFragment(plan cognition.TurnPlan, userText string, agentContext bool) ctxfrag.Fragment {
	var b strings.Builder
	if agentContext {
		b.WriteString("Turn plan (soft guidance — follow numbered steps and preferred tools when they fit; full tool schema stays available):\n")
	} else {
		b.WriteString("Turn plan (classify intent, then execute via ReAct tools — no deterministic SOP shortcut):\n")
	}
	fmt.Fprintf(&b, "- domain: %s\n- act: %s\n- mode: %s\n- reason: %s\n",
		plan.Domain, plan.Act, plan.Mode, plan.Reason)

	profileID := domaincatalog.ProbeExecutionProfile(domaincatalog.Domain(plan.Domain), plan.Act, userText)
	if profileID != "" {
		fmt.Fprintf(&b, "- execution profile: %s\n", profileID)
		if hint := domaincatalog.ProfileExecutionHint(profileID); hint != "" {
			fmt.Fprintf(&b, "- execution contract: %s\n", hint)
		}
	}

	if steps := cursorPlanSteps(plan); len(steps) > 0 {
		b.WriteString("- plan steps:\n")
		for i, step := range steps {
			fmt.Fprintf(&b, "  %d. %s\n", i+1, step)
		}
	}

	if domaincatalog.NormalizeStockAct(plan.Act) == domaincatalog.StockActMultiSymbol {
		b.WriteString(subagentOrchestratorPlanBlock())
	}
	if len(plan.Skills) > 0 {
		fmt.Fprintf(&b, "- skills: %s\n", strings.Join(plan.Skills, ", "))
	}
	if len(plan.ToolsAllow) > 0 {
		label := "preferred tools"
		if !agentContext {
			label = "allowed tools"
		}
		fmt.Fprintf(&b, "- %s: %s\n", label, strings.Join(plan.ToolsAllow, ", "))
	}
	if plan.Mode == cognition.ModeClarify && plan.ClarifyQuestion != "" {
		if agentContext {
			fmt.Fprintf(&b, "- if still ambiguous after reading session context, call clarify with: %s\n", plan.ClarifyQuestion)
		} else {
			fmt.Fprintf(&b, "- ask via clarify: %s\n", plan.ClarifyQuestion)
		}
		if len(plan.ClarifyChoices) > 0 {
			fmt.Fprintf(&b, "- choices: %s\n", strings.Join(plan.ClarifyChoices, " / "))
		}
	}
	if agentContext {
		b.WriteString("- execute the plan above; deviate only when session context or user text clearly requires it\n")
	} else {
		b.WriteString("- choose tools from the exposed schema to fulfill this turn; for multi-symbol parallel work, strongly prefer delegate_tasks over serial per-symbol calls")
	}
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 22}
}

// cursorPlanSteps returns human-readable steps for the TurnPlan fragment and SSE.
func cursorPlanSteps(plan cognition.TurnPlan) []string {
	switch plan.Domain {
	case cognition.DomainSignalProbe:
		return []string{
			"Confirm symbol and strategy/signal (clarify if missing)",
			"resolve_signal → probe_bot_signal_series",
			"Summarize buy/sell hits for the user",
		}
	case cognition.DomainBacktestRun:
		return []string{
			"Playbook: strategy-backtest-run (only backtest playbook; 回测 ≠ generate)",
			"Confirm symbol and signal (clarify if missing)",
			"run_strategy_backtest with resolved parameters",
			"Summarize PnL / log highlights",
		}
	case cognition.DomainBacktestHistory:
		return []string{"Query prior backtest logs", "Summarize relevant metrics"}
	case cognition.DomainStockAnalysis:
		switch domaincatalog.NormalizeStockAct(plan.Act) {
		case domaincatalog.StockActQuotePrice:
			return []string{"search_code (if needed)", "get_current_price", "Reply with quote snapshot"}
		case domaincatalog.StockActTechnicalAnalysis:
			return []string{"search_code (if needed)", "get_mcp_analysis", "Summarize trend/technicals"}
		case domaincatalog.StockActMultiSymbol:
			return []string{"delegate_tasks once with one task per symbol", "Merge results into a comparative reply"}
		case domaincatalog.StockActContextFollowup:
			return []string{"Reuse session symbol from WorkingState", "Continue analysis with appropriate stock tools"}
		case domaincatalog.StockActSymbolResolve:
			return []string{"search_code for the new symbol", "Continue the prior analysis intent on the new symbol"}
		default:
			return []string{"search_code (if needed)", "get_mcp_analysis or get_current_price as fit", "Answer the user's stock question"}
		}
	case cognition.DomainDCAGrid:
		if plan.Mode == cognition.ModeGather {
			return []string{"get_signal_combinations or list strategies", "Help user pick a strategy"}
		}
		return []string{
			"User asked to generate/design a DCA or Grid plan (not ordinary 回测)",
			"get_signal_combinations → pick signal_id if needed",
			"generate_dca_strategy or generate_grid_strategy",
			"Optional loopback_strategy to validate the generated plan",
		}
	case cognition.DomainBotManage:
		return []string{"List or mutate bots/reminders via trading_bot tools", "Confirm outcome to user"}
	case cognition.DomainReportLookup:
		return []string{"Query report APIs", "Summarize report content"}
	case cognition.DomainKnowledge:
		return []string{"Search knowledge base", "Answer with citations"}
	case cognition.DomainAmbiguous:
		if plan.Mode == cognition.ModeClarify {
			return []string{"Call clarify to disambiguate intent before heavy tools", "Wait for user choice, then execute"}
		}
		return nil
	default:
		return nil
	}
}

func clarifyHintFragment(plan cognition.TurnPlan) ctxfrag.Fragment {
	if plan.Mode != cognition.ModeClarify {
		return ctxfrag.StaticFragment{}
	}
	question := strings.TrimSpace(plan.ClarifyQuestion)
	if question == "" {
		return ctxfrag.StaticFragment{}
	}
	var b strings.Builder
	b.WriteString("## 分类器建议（缺槽位时优先 clarify，再执行工具）\n")
	fmt.Fprintf(&b, "- suggested_question: %s\n", question)
	if len(plan.ClarifyChoices) > 0 {
		fmt.Fprintf(&b, "- suggested_choices: %s\n", strings.Join(plan.ClarifyChoices, " / "))
	}
	if r := strings.TrimSpace(plan.Reason); r != "" {
		fmt.Fprintf(&b, "- reason: %s\n", r)
	}
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 24}
}

func subagentOrchestratorPlanBlock() string {
	return `- multi-symbol orchestration (parallel sub-agents — recommendation, not a hard tool lock):
  - When the user names 2+ distinct companies/symbols in one turn, prefer delegate_tasks ONCE with tasks[] (one self-contained task per symbol/company). Sub-agents run in parallel with isolated context; each task is a self-contained mini-turn without this chat history.
  - Each task string must include the company/symbol and what to analyze; sub-agents cannot see this chat history.
  - After delegate_tasks returns, read results[] and write the FINAL user-facing answer yourself:
    (1) one subsection per symbol with concrete price/analysis from results[];
    (2) a brief side-by-side comparison (涨跌幅、相对强弱、一句话结论);
    (3) do NOT stop at meta text like "已委派" or "子 Agent 完成" without listing both symbols' data.
  - Single-symbol or simple quote requests: call search_code / get_current_price / get_mcp_analysis directly — do not delegate.
  - You still have all stock tools available; use delegate_tasks when parallel isolation improves quality or latency, not because tools were removed.
`
}

func (l *Loop) loadPlanSkills(plan cognition.TurnPlan, userText string, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
	if l == nil || l.skillLoader == nil {
		return nil, nil
	}
	seen := map[string]struct{}{}
	matched := make([]procedural.Skill, 0, l.maxSkills+len(plan.Skills))
	appendSkill := func(sk procedural.Skill) {
		if sk.Name == "" {
			return
		}
		if _, ok := seen[sk.Name]; ok {
			return
		}
		seen[sk.Name] = struct{}{}
		matched = append(matched, sk)
	}
	for _, sk := range l.skillLoader.Match(userText, l.maxSkills) {
		appendSkill(sk)
	}
	for _, name := range plan.Skills {
		sk, ok := l.skillLoader.FindByName(name)
		if !ok {
			continue
		}
		appendSkill(sk)
	}
	if len(matched) > l.maxSkills && l.maxSkills > 0 {
		matched = matched[:l.maxSkills]
	}
	if len(matched) == 0 {
		return nil, nil
	}
	block := procedural.Format(matched)
	names := skillNames(matched)
	if block == "" {
		return nil, names
	}
	l.emitStatus("plan", fmt.Sprintf("加载 %d 个相关技能", len(matched)))
	l.emit("memory.procedural", map[string]any{
		"skills": len(matched),
		"names":  names,
		"source": "context_match",
	})
	l.recordInjectionStep(records, "context_inject", fmt.Sprintf("skills: %s", strings.Join(names, ", ")))
	return ctxfrag.ProceduralSkillFragment(block), names
}
