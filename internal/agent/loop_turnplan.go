package agent

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func agentRoutingFragment(userText string) ctxfrag.Fragment {
	var b strings.Builder
	b.WriteString("## Agent routing (ReAct)\n")
	b.WriteString("- Read session context and pick tools; no separate intent classifier runs in this turn.\n")
	b.WriteString("- If the user intent is ambiguous (e.g. quote vs analysis, which signal/strategy, analyze vs backtest), call the clarify tool with a clear question and up to 4 choices before irreversible tools.\n")
	b.WriteString("- After clarify returns user_response, continue in the same turn with that choice.\n")
	b.WriteString("- Guided skills (信号诊断、策略开发、多策略对比、生成策略档案): pick the skill id from the **Skills catalog**, call **skill_view**, then follow SKILL.md using ordinary tools (probe_bot_signal_series, diagnose_bot_signal_series, save_strategy_knowledge, …). Call **emit_workflow_card** when the skill asks for Monday UI cards.\n")
	b.WriteString(subagentOrchestratorPlanBlock())
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 22}
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
	mode := l.effectiveSkillInjection()
	switch mode {
	case procedural.SkillInjectionOff:
		return l.loadExplicitPlanSkillsOnly(plan, records)
	case procedural.SkillInjectionCatalog:
		return l.loadCatalogSkills(plan, records)
	default:
		return l.loadKeywordMatchedSkills(plan, userText, records)
	}
}

func (l *Loop) loadExplicitPlanSkillsOnly(plan cognition.TurnPlan, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
	matched := l.skillsFromPlan(plan)
	if len(matched) == 0 {
		return nil, nil
	}
	return l.formatMatchedSkills(matched, "turn_plan", records)
}

func (l *Loop) loadCatalogSkills(plan cognition.TurnPlan, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
	catalog := l.skillLoader.ChatCatalog()
	var frag ctxfrag.Fragment
	if block := procedural.FormatCatalog(catalog); block != "" {
		l.emitStatus("plan", fmt.Sprintf("注入 %d 个技能 catalog 条目", len(catalog)))
		l.emit("memory.procedural", map[string]any{
			"skills": len(catalog),
			"source": "catalog",
		})
		l.recordInjectionStep(records, "context_inject", fmt.Sprintf("skills catalog: %d entries", len(catalog)))
		frag = ctxfrag.SkillsCatalogFragment(block)
	}
	names := skillNames(l.skillsFromPlan(plan))
	return frag, names
}

func (l *Loop) loadKeywordMatchedSkills(plan cognition.TurnPlan, userText string, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
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
	for _, sk := range l.skillsFromPlan(plan) {
		appendSkill(sk)
	}
	if len(matched) > l.maxSkills && l.maxSkills > 0 {
		matched = matched[:l.maxSkills]
	}
	if len(matched) == 0 {
		return nil, nil
	}
	return l.formatMatchedSkills(matched, "context_match", records)
}

func (l *Loop) skillsFromPlan(plan cognition.TurnPlan) []procedural.Skill {
	var matched []procedural.Skill
	for _, name := range plan.Skills {
		sk, ok := l.skillLoader.FindByName(name)
		if !ok {
			continue
		}
		matched = append(matched, sk)
	}
	return matched
}

func (l *Loop) formatMatchedSkills(matched []procedural.Skill, source string, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
	block := procedural.Format(matched)
	names := skillNames(matched)
	if block == "" {
		return nil, names
	}
	l.emitStatus("plan", fmt.Sprintf("加载 %d 个相关技能", len(matched)))
	l.emit("memory.procedural", map[string]any{
		"skills": len(matched),
		"names":  names,
		"source": source,
	})
	l.recordInjectionStep(records, "context_inject", fmt.Sprintf("skills: %s", strings.Join(names, ", ")))
	return ctxfrag.ProceduralSkillFragment(block), names
}

