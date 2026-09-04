package agent

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func turnPlanFragment(plan cognition.TurnPlan) ctxfrag.Fragment {
	var b strings.Builder
	b.WriteString("Turn plan (follow this routing; do not switch domains unless the user clearly asks):\n")
	fmt.Fprintf(&b, "- domain: %s\n- act: %s\n- mode: %s\n- reason: %s\n",
		plan.Domain, plan.Act, plan.Mode, plan.Reason)
	if len(plan.Skills) > 0 {
		fmt.Fprintf(&b, "- skills: %s\n", strings.Join(plan.Skills, ", "))
	}
	if plan.Mode == cognition.ModeClarify && plan.ClarifyQuestion != "" {
		fmt.Fprintf(&b, "- ask via clarify: %s\n", plan.ClarifyQuestion)
		if len(plan.ClarifyChoices) > 0 {
			fmt.Fprintf(&b, "- choices: %s\n", strings.Join(plan.ClarifyChoices, " / "))
		}
	}
	b.WriteString("- only call tools listed for this domain; do not run backtest unless domain is backtest_run")
	return ctxfrag.StaticFragment{K: ctxfrag.KindSystemRules, Text: b.String(), Prio: 22}
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
	l.emitStatus("gate", fmt.Sprintf("加载 %d 个相关技能 (plan)", len(matched)))
	l.emit("memory.procedural", map[string]any{
		"skills": len(matched),
		"names":  names,
		"source": "turn_plan",
	})
	l.recordInjectionStep(records, "context_inject", fmt.Sprintf("plan skills: %s", strings.Join(names, ", ")))
	return ctxfrag.ProceduralSkillFragment(block), names
}
