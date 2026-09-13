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
		b.WriteString("Turn plan telemetry (observability only — you decide tools via ReAct + skills):\n")
	} else {
		b.WriteString("Turn plan (classify intent, then execute via ReAct tools — no deterministic SOP shortcut):\n")
	}
	fmt.Fprintf(&b, "- domain: %s\n- act: %s\n- mode: %s\n- reason: %s\n",
		plan.Domain, plan.Act, plan.Mode, plan.Reason)
	if !agentContext {
		if profileID := domaincatalog.ProbeExecutionProfile(domaincatalog.Domain(plan.Domain), plan.Act, userText); profileID != "" {
			fmt.Fprintf(&b, "- execution profile: %s\n", profileID)
			if hint := domaincatalog.ProfileExecutionHint(profileID); hint != "" {
				fmt.Fprintf(&b, "- execution contract: %s\n", hint)
			}
		}
	}
	if domaincatalog.NormalizeStockAct(plan.Act) == domaincatalog.StockActMultiSymbol {
		b.WriteString(subagentOrchestratorPlanBlock())
	}
	if len(plan.Skills) > 0 {
		fmt.Fprintf(&b, "- skills: %s\n", strings.Join(plan.Skills, ", "))
	}
	if !agentContext && len(plan.ToolsAllow) > 0 {
		fmt.Fprintf(&b, "- allowed tools: %s\n", strings.Join(plan.ToolsAllow, ", "))
	}
	if !agentContext && plan.Mode == cognition.ModeClarify && plan.ClarifyQuestion != "" {
		fmt.Fprintf(&b, "- ask via clarify: %s\n", plan.ClarifyQuestion)
		if len(plan.ClarifyChoices) > 0 {
			fmt.Fprintf(&b, "- choices: %s\n", strings.Join(plan.ClarifyChoices, " / "))
		}
	}
	b.WriteString("- choose tools from the exposed schema to fulfill this turn; for multi-symbol parallel work, strongly prefer delegate_tasks over serial per-symbol calls")
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 22}
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
	b.WriteString("## 分类器建议（仅供参考，由你决定是否 clarify）\n")
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
