package cognition

import (
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func enrichStockAnalysisAct(plan TurnPlan, in PlanInput) TurnPlan {
	if plan.Domain != DomainStockAnalysis {
		return plan
	}
	plan.Act = domaincatalog.RefineStockAnalysisAct(in.UserText, domaincatalog.Domain(in.LastDomain))
	return plan
}
