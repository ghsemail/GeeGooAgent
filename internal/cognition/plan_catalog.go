package cognition

import (
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func planForDomain(d Domain) TurnPlan {
	return turnPlanFromSpec(domaincatalog.MakePlan(domaincatalog.Domain(d)))
}

func turnPlanFromSpec(spec domaincatalog.PlanSpec) TurnPlan {
	return TurnPlan{
		Domain:          Domain(spec.Domain),
		Act:             spec.Act,
		Mode:            Mode(spec.Mode),
		Confidence:      spec.Confidence,
		Reason:          spec.Reason,
		Skills:          append([]string(nil), spec.Skills...),
		ToolsAllow:      append([]string(nil), spec.ToolsAllow...),
		ClarifyQuestion: spec.ClarifyQuestion,
		ClarifyChoices:  append([]string(nil), spec.ClarifyChoices...),
	}
}
