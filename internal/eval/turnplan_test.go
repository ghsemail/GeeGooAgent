package eval_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestDefaultTurnPlanSuitePasses(t *testing.T) {
	results := eval.RunTurnPlanSuite(eval.DefaultTurnPlanSuite())
	if !eval.AllTurnPlanPass(results) {
		for _, r := range results {
			if !r.Passed {
				t.Errorf("%s: %s", r.TurnID, r.Detail)
			}
		}
	}
}

func TestTurnPlanSuiteRejectsWrongDomain(t *testing.T) {
	suite := eval.DefaultTurnPlanSuite()
	suite.Turns[0].ExpectDomain = "backtest_run"
	results := eval.RunTurnPlanSuite(suite)
	if eval.AllTurnPlanPass(results) {
		t.Fatal("expected failure with wrong domain")
	}
}
