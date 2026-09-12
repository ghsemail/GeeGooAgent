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

func turnPlanFragment(plan cognition.TurnPlan) ctxfrag.Fragment {
	var b strings.Builder
	b.WriteString("Turn plan (classify intent, then execute via ReAct tools — no deterministic SOP shortcut):\n")
	fmt.Fprintf(&b, "- domain: %s\n- act: %s\n- mode: %s\n- reason: %s\n",
		plan.Domain, plan.Act, plan.Mode, plan.Reason)
	if profileID := domaincatalog.ExecutionProfileFor(domaincatalog.Domain(plan.Domain), plan.Act); profileID != "" {
		fmt.Fprintf(&b, "- execution profile: %s\n", profileID)
		if hint := domaincatalog.ProfileExecutionHint(profileID); hint != "" {
			fmt.Fprintf(&b, "- execution contract: %s\n", hint)
		}
	}
	if domaincatalog.NormalizeStockAct(plan.Act) == domaincatalog.StockActMultiSymbol {
		b.WriteString(subagentOrchestratorPlanBlock())
	}
	if len(plan.Skills) > 0 {
		fmt.Fprintf(&b, "- skills: %s\n", strings.Join(plan.Skills, ", "))
	}
	if len(plan.ToolsAllow) > 0 {
		fmt.Fprintf(&b, "- allowed tools: %s\n", strings.Join(plan.ToolsAllow, ", "))
	}
	if plan.Domain == cognition.DomainSignalProbe && plan.Mode == cognition.ModeExecute {
		b.WriteString(signalProbeExecutePlanBlock())
	}
	if plan.Mode == cognition.ModeClarify && plan.ClarifyQuestion != "" {
		fmt.Fprintf(&b, "- ask via clarify: %s\n", plan.ClarifyQuestion)
		if len(plan.ClarifyChoices) > 0 {
			fmt.Fprintf(&b, "- choices: %s\n", strings.Join(plan.ClarifyChoices, " / "))
		}
	}
	b.WriteString("- choose tools from the exposed schema to fulfill this turn; for multi-symbol parallel work, strongly prefer delegate_tasks over serial per-symbol calls")
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 22}
}

func signalProbeExecutePlanBlock() string {
	return `- signal probe continuation (same session):
  - If user is swapping strategy / signal only, keep the same stock code from session; do NOT re-ask analyze vs probe vs backtest.
  - When strategy is missing, clarify with catalog strategy names only (max 4 choices).
  - After clarify returns a strategy name, you MUST call probe_bot_signal_series in THIS turn before the final reply (months_back=3 unless user specified otherwise; mirror sell_signal to buy_signal for single-indicator rules).
`
}

func subagentOrchestratorPlanBlock() string {
	return `- multi-symbol orchestration (Cursor Task-style — recommendation, not a hard tool lock):
  - When the user names 2+ distinct companies/symbols in one turn, prefer delegate_tasks ONCE with tasks[] (one self-contained task per symbol/company). Sub-agents run in parallel with isolated context, like multiple Task calls in one Cursor turn.
  - Each task string must include the company/symbol and what to analyze; sub-agents cannot see this chat history.
  - After delegate_tasks returns, read results[] and write the FINAL user-facing answer yourself:
    (1) one subsection per symbol with concrete price/analysis from results[];
    (2) a brief side-by-side comparison (涨跌幅、相对强弱、一句话结论);
    (3) do NOT stop at meta text like "已委派" or "子 Agent 完成" without listing both symbols' data.
  - Single-symbol or simple quote requests: call search_code / get_current_price / get_mcp_analysis directly — do not delegate.
  - You still have all stock tools available; use delegate_tasks when parallel isolation improves quality or latency, not because tools were removed.
`
}

func (l *Loop) loadPlanSkills(plan cognition.TurnPlan, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
	if l == nil || l.skillLoader == nil || len(plan.Skills) == 0 {
		return nil, nil
	}
	matched := make([]procedural.Skill, 0, len(plan.Skills))
	for _, name := range plan.Skills {
		sk, ok := l.skillLoader.FindByName(name)
		if !ok {
			continue
		}
		matched = append(matched, sk)
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
		"source": "turn_plan",
	})
	l.recordInjectionStep(records, "context_inject", fmt.Sprintf("plan skills: %s", strings.Join(names, ", ")))
	return ctxfrag.ProceduralSkillFragment(block), names
}
