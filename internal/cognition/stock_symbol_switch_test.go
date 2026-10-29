package cognition

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func TestApplyStockSymbolSwitchAct(t *testing.T) {
	analyzePlan := func() TurnPlan {
		p := planForDomain(DomainStockAnalysis)
		p.Mode = ModeGather
		p.Act = domaincatalog.StockActAnalyze
		p.Reason = "llm: test"
		return p
	}
	techPlan := func() TurnPlan {
		p := analyzePlan()
		p.Act = domaincatalog.StockActTechnicalAnalysis
		return p
	}

	t.Run("last domain promote analyze to symbol_resolve", func(t *testing.T) {
		in := PlanInput{UserText: "再帮我分析一下贵州茅台", LastDomain: DomainStockAnalysis}
		got := applyStockSymbolSwitchAct(in, analyzePlan())
		if got.Act != domaincatalog.StockActSymbolResolve {
			t.Fatalf("act=%s want symbol_resolve", got.Act)
		}
	})

	t.Run("no last domain keeps analyze", func(t *testing.T) {
		in := PlanInput{UserText: "帮我分析一下贵州茅台"}
		got := applyStockSymbolSwitchAct(in, analyzePlan())
		if got.Act != domaincatalog.StockActAnalyze {
			t.Fatalf("act=%s want analyze", got.Act)
		}
	})

	t.Run("technical chain keeps technical_analysis", func(t *testing.T) {
		in := PlanInput{UserText: "再帮我分析一下腾讯的K线图", LastDomain: DomainStockAnalysis}
		got := applyStockSymbolSwitchAct(in, techPlan())
		if got.Act != domaincatalog.StockActTechnicalAnalysis {
			t.Fatalf("act=%s want technical_analysis", got.Act)
		}
	})

	t.Run("pronoun follow-up not promoted", func(t *testing.T) {
		in := PlanInput{UserText: "它最近的信号趋势怎么样", LastDomain: DomainStockAnalysis}
		got := applyStockSymbolSwitchAct(in, analyzePlan())
		if got.Act != domaincatalog.StockActAnalyze {
			t.Fatalf("act=%s want analyze unchanged", got.Act)
		}
	})
}

func TestIntentPlannerStockSymbolSwitchLift(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"stock_analysis","mode":"gather","act":"analyze","confidence":0.9,"reason":"general analysis"}`}
	p := IntentPlanner{LLM: mock}

	got := p.Plan(PlanInput{UserText: "再帮我分析一下贵州茅台", LastDomain: DomainStockAnalysis})
	if got.Act != domaincatalog.StockActSymbolResolve {
		t.Fatalf("with LastDomain act=%s want symbol_resolve", got.Act)
	}

	got = p.Plan(PlanInput{UserText: "帮我分析一下贵州茅台"})
	if got.Act != domaincatalog.StockActAnalyze {
		t.Fatalf("without LastDomain act=%s want analyze", got.Act)
	}

	mock.body = `{"domain":"stock_analysis","mode":"gather","act":"technical_analysis","confidence":0.9,"reason":"kline"}`
	got = p.Plan(PlanInput{UserText: "再帮我分析一下腾讯的K线图", LastDomain: DomainStockAnalysis})
	if got.Act != domaincatalog.StockActTechnicalAnalysis {
		t.Fatalf("technical chain act=%s want technical_analysis", got.Act)
	}
}
