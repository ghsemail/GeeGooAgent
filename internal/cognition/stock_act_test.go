package cognition_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func TestRulePlannerStockActs(t *testing.T) {
	p := cognition.RulePlanner{}
	cases := []struct {
		msg  string
		last cognition.Domain
		act  string
	}{
		{msg: "帮我查一下腾讯控股现在的股价", act: domaincatalog.StockActQuotePrice},
		{msg: "再帮我看看腾讯的技术面和K线图", last: cognition.DomainStockAnalysis, act: domaincatalog.StockActTechnicalAnalysis},
		{msg: "它最近走势怎么样", last: cognition.DomainStockAnalysis, act: domaincatalog.StockActContextFollowup},
		{msg: "不聊中际旭创了，帮我分析一下贵州茅台", last: cognition.DomainStockAnalysis, act: domaincatalog.StockActSymbolResolve},
	}
	for _, tc := range cases {
		plan := p.Plan(cognition.PlanInput{UserText: tc.msg, LastDomain: tc.last})
		if plan.Domain != cognition.DomainStockAnalysis {
			t.Fatalf("%q domain=%s", tc.msg, plan.Domain)
		}
		if plan.Act != tc.act {
			t.Fatalf("%q act=%s want %s", tc.msg, plan.Act, tc.act)
		}
	}
}
