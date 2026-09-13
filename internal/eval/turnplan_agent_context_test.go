package eval_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

// Self-test: 3 plan-only eval turns (classify observability, no legacy routing gates).
func TestAgentContextPlanOnlySampleCases(t *testing.T) {
	suite := eval.TurnPlanSuite{
		PlanOnly: true,
		Turns: []eval.TurnPlanTurn{
			{
				ID: "signal_probe_direct", Message: "帮我看看中际旭创有没有买卖点",
				ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: false,
			},
			{
				ID: "backtest_explicit", Message: "帮我用SAR加MACD回测一下小米",
				ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			},
			{
				ID: "stock_price", Message: "帮我查一下腾讯控股现在的股价",
				ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectAct: "quote_price", ExpectSOP: false,
			},
		},
	}
	report := eval.RunTurnPlanReport(suite, eval.DefaultTurnPlanPlanner())
	if report.Failed != 0 || !report.AllPass {
		for _, r := range report.Results {
			if !r.Passed {
				t.Fatalf("turn %s failed: %s", r.TurnID, r.Detail)
			}
		}
		t.Fatalf("sample eval: passed=%d failed=%d", report.Passed, report.Failed)
	}
}
