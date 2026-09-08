package eval

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

// TurnPlanResult is the outcome of one turn_plan eval assertion.
type TurnPlanResult struct {
	TurnID string
	Passed bool
	Detail string
}

// RunTurnPlanSuite exercises the given Planner against a suite (no tools).
func RunTurnPlanSuite(suite TurnPlanSuite, planner cognition.Planner) []TurnPlanResult {
	if planner == nil {
		planner = DefaultTurnPlanPlanner()
	}
	out := make([]TurnPlanResult, 0, len(suite.Turns))
	for _, turn := range suite.Turns {
		out = append(out, checkTurnPlan(planner, turn))
	}
	return out
}

func checkTurnPlan(planner cognition.Planner, turn TurnPlanTurn) TurnPlanResult {
	plan := planner.Plan(cognition.PlanInput{
		UserText:   turn.Message,
		LastDomain: cognition.Domain(turn.LastDomain),
	})
	res := TurnPlanResult{TurnID: turn.ID, Passed: true}

	var problems []string
	if string(plan.Domain) != turn.ExpectDomain {
		problems = append(problems, fmt.Sprintf("domain=%s want %s", plan.Domain, turn.ExpectDomain))
	}
	if string(plan.Mode) != turn.ExpectMode {
		problems = append(problems, fmt.Sprintf("mode=%s want %s", plan.Mode, turn.ExpectMode))
	}
	if turn.ExpectAct != "" && plan.Act != turn.ExpectAct {
		problems = append(problems, fmt.Sprintf("act=%s want %s", plan.Act, turn.ExpectAct))
	}
	if plan.ShouldRunDomainSOP() != turn.ExpectSOP {
		problems = append(problems, fmt.Sprintf("sop=%v want %v", plan.ShouldRunDomainSOP(), turn.ExpectSOP))
	}
	for _, tool := range turn.ForbidTools {
		if containsTool(plan.ToolsAllow, tool) {
			problems = append(problems, fmt.Sprintf("forbid tool %s present", tool))
		}
	}
	for _, tool := range turn.RequireTools {
		if !containsTool(plan.ToolsAllow, tool) {
			problems = append(problems, fmt.Sprintf("missing tool %s", tool))
		}
	}
	if len(problems) > 0 {
		res.Passed = false
		res.Detail = strings.Join(problems, "; ")
	} else {
		detail := fmt.Sprintf("%s/%s sop=%v", plan.Domain, plan.Mode, plan.ShouldRunDomainSOP())
		if plan.Act != "" {
			detail += " act=" + plan.Act
		}
		res.Detail = detail
	}
	return res
}

func containsTool(allow []string, name string) bool {
	for _, t := range allow {
		if t == name {
			return true
		}
	}
	return false
}

// AllTurnPlanPass reports whether every result passed.
func AllTurnPlanPass(results []TurnPlanResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
