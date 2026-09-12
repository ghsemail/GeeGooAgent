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
		{Name: "delegate_tasks"},
		{Name: "create_dca_bot"},
	}, plan)
	names := map[string]bool{}
	for _, s := range filtered {
		names[s.Name] = true
	}
	if !names["search_code"] || !names["clarify"] || !names["delegate_tasks"] {
		t.Fatalf("filtered=%v", names)
	}
	if names["run_strategy_backtest"] || names["create_dca_bot"] {
		t.Fatalf("backtest/bot tools leaked: %v", names)
	}
}

func TestFilterSchemasOmitsDelegateOnTalk(t *testing.T) {
	plan := TurnPlan{Domain: DomainChat, Mode: ModeTalk, ToolsAllow: []string{"search_code"}}
	filtered := FilterSchemas([]llm.ToolSchema{
		{Name: "search_code"},
		{Name: "delegate_tasks"},
		{Name: "clarify"},
	}, plan)
	names := map[string]bool{}
	for _, s := range filtered {
		names[s.Name] = true
	}
	if !names["search_code"] || !names["clarify"] {
		t.Fatalf("filtered=%v", names)
	}
	if names["delegate_tasks"] {
		t.Fatal("talk mode must not expose delegate_tasks")
	}
}

type classifyMock struct {
	calls  int
	body   string
	bodies []string
}

func (m *classifyMock) Model() string { return "gate-mock" }

func (m *classifyMock) Chat(_ context.Context, _ []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	m.calls++
	if m != nil && len(m.bodies) > 0 {
		idx := m.calls - 1
		if idx >= 0 && idx < len(m.bodies) {
			return &llm.Response{Content: m.bodies[idx]}, nil
		}
	}
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

	mock.body = `{"domain":"stock_analysis","mode":"gather","act":"multi_symbol_delegate","confidence":0.9,"reason":"two symbols parallel"}`
	got = p.Plan(PlanInput{UserText: "请帮我分析下腾讯和阿里巴巴最近的股价"})
	if got.Act != domaincatalog.StockActMultiSymbol {
		t.Fatalf("act=%s want multi_symbol_delegate", got.Act)
	}
	if domaincatalog.ExecutionProfileFor(domaincatalog.DomainStockAnalysis, got.Act) != domaincatalog.ProfileSubagentMultiStock {
		t.Fatalf("expected subagent execution profile for multi_symbol_delegate")
	}
	if len(got.ToolsAllow) != 1 || got.ToolsAllow[0] != "delegate_tasks" {
		t.Fatalf("multi_symbol_delegate orchestrator ToolsAllow=%v want [delegate_tasks]", got.ToolsAllow)
	}

	mock.body = `{"domain":"stock_analysis","mode":"gather","act":"technical_analysis","symbol_count":2,"confidence":0.9,"reason":"two companies"}`
	got = p.Plan(PlanInput{UserText: "请帮我分析下腾讯和阿里巴巴最近的股价"})
	if got.Act != domaincatalog.StockActMultiSymbol {
		t.Fatalf("symbol_count=2 should promote act to multi_symbol_delegate, got %s", got.Act)
	}

	mock = &classifyMock{bodies: []string{
		`{"domain":"stock_analysis","mode":"gather","act":"quote_price","symbol_count":0,"confidence":0.9,"reason":"price quote"}`,
		`{"symbol_count":2,"reason":"腾讯和阿里巴巴"}`,
	}}
	p = IntentPlanner{LLM: mock}
	got = p.Plan(PlanInput{UserText: "请帮我分析下腾讯和阿里巴巴最近的股价"})
	if got.Act != domaincatalog.StockActMultiSymbol {
		t.Fatalf("symbol_count refine should promote quote_price to multi_symbol_delegate, got %s", got.Act)
	}
	if mock.calls != 2 {
		t.Fatalf("expected classify + symbol_count calls, got %d", mock.calls)
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

func TestIntentPlannerSignalUsageClarifyTemplate(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"ambiguous","mode":"clarify","clarify":"signal_usage","confidence":0.7,"reason":"bare macd usage"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "这个MACD信号平时该怎么用比较好"})
	if got.ClarifyQuestion != domaincatalog.SignalUsageClarifyQuestion {
		t.Fatalf("question=%q", got.ClarifyQuestion)
	}
	if len(got.ClarifyChoices) != 2 || got.ClarifyChoices[0] != "SAR信号搭配MACD直方图趋势" {
		t.Fatalf("choices=%v", got.ClarifyChoices)
	}
}

func TestIntentPlannerCompoundStepsClarifyTemplate(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"ambiguous","mode":"clarify","clarify":"compound_steps","confidence":0.7,"reason":"compound"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "帮我把中际旭创分析一下，然后再跑个回测看看效果"})
	if got.ClarifyQuestion != domaincatalog.CompoundStepClarifyQuestion {
		t.Fatalf("question=%q", got.ClarifyQuestion)
	}
	if len(got.ClarifyChoices) != 2 || got.ClarifyChoices[0] != "先只做分析" {
		t.Fatalf("choices=%v", got.ClarifyChoices)
	}
}

func TestIntentPlannerSignalChoiceMapsToKnowledge(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"chat","mode":"talk","confidence":0.9,"reason":"wrong"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "SAR信号搭配MACD直方图趋势", LastDomain: DomainAmbiguous})
	if got.Domain != DomainKnowledge {
		t.Fatalf("choice should map to knowledge, got %s/%s", got.Domain, got.Mode)
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

func TestIntentPlannerCompoundStepsNotForcedToBacktest(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"ambiguous","mode":"clarify","clarify":"compound_steps","confidence":0.7,"reason":"compound"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "帮我把中际旭创分析一下，然后再跑个回测看看效果"})
	if got.Domain != DomainAmbiguous || got.Mode != ModeClarify {
		t.Fatalf("compound analyze+backtest should stay ambiguous/clarify, got %s/%s", got.Domain, got.Mode)
	}
}

func TestIntentPlannerForcesBacktestRunWithVerb(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"chat","mode":"talk","confidence":0.9,"reason":"misroute"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "帮我用这些策略回测一下", LastDomain: DomainDCAGrid})
	if got.Domain != DomainBacktestRun || got.Mode != ModeExecute {
		t.Fatalf("backtest verb should force backtest_run/execute, got %s/%s", got.Domain, got.Mode)
	}
}

func TestIntentPlannerBacktestVerbOverridesStickyDomain(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"chat","mode":"talk","confidence":0.9,"reason":"misroute"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "接着帮小米跑个回测", LastDomain: DomainStockAnalysis})
	if got.Domain != DomainBacktestRun || got.Mode != ModeExecute {
		t.Fatalf("backtest verb must override sticky stock_analysis, got %s/%s", got.Domain, got.Mode)
	}
}

func TestIntentPlannerBacktestRunOmitsLoopbackTools(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"backtest_run","mode":"execute","confidence":0.9,"reason":"backtest"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "帮我用这些策略回测一下", LastDomain: DomainDCAGrid})
	if got.Domain != DomainBacktestRun || got.Mode != ModeExecute {
		t.Fatalf("domain=%s mode=%s", got.Domain, got.Mode)
	}
	if !containsStr(got.ToolsAllow, "run_strategy_backtest") {
		t.Fatalf("backtest_run must allow run_strategy_backtest: %v", got.ToolsAllow)
	}
	for _, forbidden := range []string{"loopback_strategy", "generate_dca_strategy", "generate_grid_strategy"} {
		if containsStr(got.ToolsAllow, forbidden) {
			t.Fatalf("backtest_run must not allow %s: %v", forbidden, got.ToolsAllow)
		}
	}
}

func TestIntentPlannerDCAGridGatherOmitsBacktestTool(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"dca_grid","mode":"gather","confidence":0.9,"reason":"list strategies"}`}
	p := IntentPlanner{LLM: mock}
	got := p.Plan(PlanInput{UserText: "帮我看看哪些策略适合腾讯", LastDomain: DomainStockAnalysis})
	if got.Domain != DomainDCAGrid || got.Mode != ModeGather {
		t.Fatalf("domain=%s mode=%s", got.Domain, got.Mode)
	}
	if containsStr(got.ToolsAllow, "run_strategy_backtest") {
		t.Fatalf("gather mode must not expose run_strategy_backtest: %v", got.ToolsAllow)
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
