package cognition

import "strings"

// PlanAfterPresetClarify returns the TurnPlan to continue after the user picks a
// preset clarify option. It avoids re-classifying long choice text as a new turn.
func PlanAfterPresetClarify(parent TurnPlan, answer string) TurnPlan {
	answer = strings.TrimSpace(answer)
	plan := parent
	if d, act, ok := mapClarifyChoice(answer); ok {
		if parent.Domain == DomainAmbiguous || d == parent.Domain {
			plan = planForDomain(d)
			if act != "" {
				plan.Act = act
			}
		}
	}
	if parent.Mode == ModeClarify {
		switch plan.Domain {
		case DomainDCAGrid:
			plan.Mode = ModeGather
		case DomainStockAnalysis:
			plan.Mode = ModeGather
		default:
			plan.Mode = ModeExecute
		}
	}
	plan.ClarifyQuestion = ""
	plan.ClarifyChoices = nil
	if answer != "" {
		plan.Reason = "preset clarify answered: " + truncateClarifyReason(answer, 80)
	}
	return applyPlanToolPolicies(plan)
}

func truncateClarifyReason(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
