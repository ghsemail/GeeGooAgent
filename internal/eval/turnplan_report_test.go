package eval_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestRunTurnPlanReportDefaultSuite(t *testing.T) {
	report := eval.RunTurnPlanReport(eval.DefaultTurnPlanSuite())
	if !report.AllPass {
		t.Fatalf("expected all pass, got %+v", report)
	}
	if report.Total != len(eval.DefaultTurnPlanSuite().Turns) {
		t.Fatalf("total=%d want %d", report.Total, len(eval.DefaultTurnPlanSuite().Turns))
	}
	if report.Results[0].Message == "" {
		t.Fatal("expected message on first result")
	}
}

func TestSuiteFromOptions(t *testing.T) {
	opts := map[string]any{
		"category":  "turn_plan",
		"plan_only": true,
		"turns": []any{
			map[string]any{
				"id":            "chat_definition",
				"message":       "MACD 指标是什么意思",
				"expect_domain": "chat",
				"expect_mode":   "talk",
				"expect_sop":    false,
			},
		},
	}
	suite, err := eval.SuiteFromOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(suite.Turns) != 1 || suite.Turns[0].ID != "chat_definition" {
		t.Fatalf("suite=%+v", suite)
	}
}
