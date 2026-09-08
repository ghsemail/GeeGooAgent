package cognition

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestFilterSchemasIntersectsAllowList(t *testing.T) {
	plan := IntentPlanner{LLM: &ClassifyFixtureProvider{
		ByMessage: map[string]string{
			"腾讯现在怎么样": FormatClassifyJSON("stock_analysis", "gather", "analyze", "fixture"),
		},
	}}.Plan(PlanInput{UserText: "腾讯现在怎么样"})
	filtered := FilterSchemas([]llm.ToolSchema{
		{Name: "search_code"},
		{Name: "run_strategy_backtest"},
		{Name: "clarify"},
		{Name: "create_dca_bot"},
	}, plan)
	names := map[string]bool{}
	for _, s := range filtered {
		names[s.Name] = true
	}
	if !names["search_code"] || !names["clarify"] {
		t.Fatalf("filtered=%v", names)
	}
	if names["run_strategy_backtest"] || names["create_dca_bot"] {
		t.Fatalf("backtest/bot tools leaked: %v", names)
	}
}

type classifyMock struct {
	calls int
	body  string
}

func (m *classifyMock) Model() string { return "gate-mock" }

func (m *classifyMock) Chat(_ context.Context, _ []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	m.calls++
	return &llm.Response{Content: m.body}, nil
}

func TestIntentPlannerPrefersLLMWhenAvailable(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"stock_analysis","mode":"gather","confidence":0.8,"reason":"indicator mention"}`}
	p := IntentPlanner{LLM: mock}

	got := p.Plan(PlanInput{UserText: "腾讯现在怎么样"})
	if got.Domain != DomainStockAnalysis {
		t.Fatalf("analysis domain=%s", got.Domain)
	}
	if mock.calls != 1 {
		t.Fatalf("LLM should classify every turn when available, calls=%d", mock.calls)
	}

	mock.calls = 0
	mock.body = `{"domain":"ambiguous","mode":"clarify","confidence":0.7,"reason":"bare macd"}`
	got = p.Plan(PlanInput{UserText: "MACD"})
	if mock.calls != 1 {
		t.Fatalf("LLM should classify ambiguous turns, calls=%d", mock.calls)
	}
	if got.Domain != DomainAmbiguous || got.Mode != ModeClarify {
		t.Fatalf("MACD should be clarify, got %s/%s", got.Domain, got.Mode)
	}
}

func TestIntentPlannerSkipsLLMOnClarifyChoice(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"chat","mode":"talk","confidence":0.9,"reason":"wrong"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "测买卖点", LastDomain: DomainAmbiguous})
	if mock.calls != 0 {
		t.Fatalf("clarify choice must not call LLM, calls=%d", mock.calls)
	}
	if got.Domain != DomainSignalProbe || got.Mode != ModeExecute {
		t.Fatalf("choice should map to probe, got %s/%s", got.Domain, got.Mode)
	}
}

func TestIntentPlannerRejectsBacktestWithoutVerb(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"backtest_run","mode":"execute","confidence":0.99,"reason":"signals"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "这个信号怎么样"})
	if got.ShouldRunBacktestPlaybook() || got.Domain == DomainBacktestRun {
		t.Fatalf("must not accept backtest_run without verb: %+v", got)
	}
}

func TestIntentPlannerStockActFromLLM(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"stock_analysis","mode":"gather","act":"quote_price","confidence":0.9,"reason":"price quote"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "帮我查一下腾讯控股现在的股价"})
	if got.Act != "quote_price" {
		t.Fatalf("act=%s want quote_price", got.Act)
	}

	mock.body = `{"domain":"stock_analysis","mode":"gather","act":"context_followup","confidence":0.9,"reason":"pronoun follow-up"}`
	got = p.Plan(PlanInput{UserText: "它最近走势怎么样", LastDomain: DomainStockAnalysis})
	if got.Act != "context_followup" {
		t.Fatalf("act=%s want context_followup", got.Act)
	}
}

func TestIntentPlannerStockClarifyChoice(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"chat","mode":"talk","confidence":0.9,"reason":"wrong"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "只要当前价", LastDomain: DomainAmbiguous})
	if got.Domain != DomainStockAnalysis || got.Act != "quote_price" {
		t.Fatalf("choice should map to quote_price, got %s/%s act=%s", got.Domain, got.Mode, got.Act)
	}

	got = p.Plan(PlanInput{UserText: "分析价格走势", LastDomain: DomainAmbiguous})
	if got.Domain != DomainStockAnalysis || got.Act != "technical_analysis" {
		t.Fatalf("choice should map to technical_analysis, got %s act=%s", got.Domain, got.Act)
	}
}

func TestIntentPlannerStockQuoteClarifyTemplate(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"ambiguous","mode":"clarify","clarify":"stock_quote","confidence":0.6,"reason":"quote vs analysis"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "腾讯股价怎么样"})
	if got.Mode != ModeClarify {
		t.Fatalf("mode=%s", got.Mode)
	}
	if got.ClarifyQuestion != domaincatalog.StockPriceClarifyQuestion {
		t.Fatalf("question=%q", got.ClarifyQuestion)
	}
	if len(got.ClarifyChoices) != 2 || got.ClarifyChoices[0] != "只要当前价" {
		t.Fatalf("choices=%v", got.ClarifyChoices)
	}
}

func TestIntentPlannerNilLLMUsesFallback(t *testing.T) {
	p := IntentPlanner{}
	got := p.Plan(PlanInput{UserText: "MACD"})
	if got.Domain != DomainChat {
		t.Fatalf("nil LLM should use conservative fallback, got %s", got.Domain)
	}
}

func TestIntentPlannerFallbackInheritsStickyDomain(t *testing.T) {
	p := IntentPlanner{}
	got := p.Plan(PlanInput{UserText: "它最近走势怎么样", LastDomain: DomainStockAnalysis})
	if got.Domain != DomainStockAnalysis {
		t.Fatalf("fallback should inherit sticky domain, got %s", got.Domain)
	}
}

func TestIntentPlannerStickySessionOverridesChatMisroute(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"chat","mode":"talk","confidence":0.8,"reason":"misroute"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "它最近走势怎么样", LastDomain: DomainStockAnalysis})
	if got.Domain != DomainStockAnalysis || got.Act != "context_followup" {
		t.Fatalf("sticky override got %s/%s act=%s", got.Domain, got.Mode, got.Act)
	}
}

func containsStr(items []string, want string) bool {
	for _, s := range items {
		if s == want {
			return true
		}
	}
	return false
}
