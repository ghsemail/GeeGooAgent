package cognition

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

func TestRulePlannerRoutesAcrossDomains(t *testing.T) {
	p := RulePlanner{}
	cases := []struct {
		msg    string
		domain Domain
		mode   Mode
		forbid []string
		want   string
	}{
		{msg: "腾讯现在怎么样", domain: DomainStockAnalysis, mode: ModeGather, want: "search_code"},
		{msg: "MACD 是什么", domain: DomainChat, mode: ModeTalk, forbid: []string{"run_strategy_backtest", "get_index_signals"}},
		{msg: "有没有买卖点", domain: DomainSignalProbe, mode: ModeExecute, want: "probe_bot_signal_series"},
		{msg: "帮我回测小米 SAR+MACD", domain: DomainBacktestRun, mode: ModeExecute, want: "run_strategy_backtest"},
		{msg: "我的网格机器人呢", domain: DomainBotManage, mode: ModeGather, want: "list_grid_bots"},
		{msg: "创建 bot", domain: DomainBotManage, mode: ModeExecute, want: "create_dca_bot"},
		{msg: "今天盘前写了什么", domain: DomainReportLookup, mode: ModeGather, want: "get_stock_premarket_reports"},
		{msg: "按知识库讲 4H MACD", domain: DomainKnowledge, mode: ModeGather, want: "search_knowledge"},
		{msg: "这个信号怎么样", domain: DomainAmbiguous, mode: ModeClarify},
		{msg: "帮我做 dca 定投回测", domain: DomainDCAGrid, mode: ModeExecute, want: "generate_dca_strategy"},
		{msg: "上次回测结果", domain: DomainBacktestHistory, mode: ModeGather, want: "list_strategy_backtest_logs"},
		{msg: "有什么新闻", domain: DomainNews, mode: ModeGather, want: "fetch_market_news"},
		{msg: "把分析写成报告", domain: DomainReportWrite, mode: ModeExecute, want: "create_stock_intraday_report"},
		{msg: "改一下定制信号参数", domain: DomainCustomSignal, mode: ModeGather, want: "get_custom_signal"},
		{msg: "加一个 EMA 模板", domain: DomainPromptAdmin, mode: ModeExecute, want: "add_single_prompt_template"},
		{msg: "准吗", domain: DomainChat, mode: ModeTalk, forbid: []string{"run_strategy_backtest"}},
		{msg: "MACD", domain: DomainAmbiguous, mode: ModeClarify, forbid: []string{"run_strategy_backtest"}},
		{msg: "中际旭创呢", domain: DomainStockAnalysis, mode: ModeGather, want: "search_code"},
		{msg: "分析一下中际旭创再回测", domain: DomainAmbiguous, mode: ModeClarify},
	}
	for _, tc := range cases {
		plan := p.Plan(PlanInput{UserText: tc.msg})
		if plan.Domain != tc.domain || plan.Mode != tc.mode {
			t.Fatalf("%q: domain=%s mode=%s want %s/%s (%s)",
				tc.msg, plan.Domain, plan.Mode, tc.domain, tc.mode, plan.Reason)
		}
		if tc.want != "" && !containsStr(plan.ToolsAllow, tc.want) {
			t.Fatalf("%q: tools=%v missing %s", tc.msg, plan.ToolsAllow, tc.want)
		}
		for _, bad := range tc.forbid {
			if containsStr(plan.ToolsAllow, bad) {
				t.Fatalf("%q: tools should not include %s", tc.msg, bad)
			}
		}
		if tc.domain == DomainBacktestRun && !plan.ShouldRunDomainSOP() {
			t.Fatalf("%q: expected domain SOP", tc.msg)
		}
		if tc.domain != DomainBacktestRun && tc.domain != DomainStockAnalysis && tc.domain != DomainSignalProbe && plan.ShouldRunDomainSOP() {
			t.Fatalf("%q: must not run domain SOP", tc.msg)
		}
	}
}

func TestFilterSchemasIntersectsAllowList(t *testing.T) {
	plan := RulePlanner{}.Plan(PlanInput{UserText: "腾讯现在怎么样"})
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

func TestRulePlannerFollowsLastDomainAndClarifyChoices(t *testing.T) {
	p := RulePlanner{}
	cases := []struct {
		msg      string
		last     Domain
		domain   Domain
		mode     Mode
		playbook bool
	}{
		{msg: "换成贵州茅台", last: DomainStockAnalysis, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "MACD", last: DomainStockAnalysis, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "个股/指标分析", last: DomainAmbiguous, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "测买卖点", last: DomainAmbiguous, domain: DomainSignalProbe, mode: ModeExecute},
		{msg: "跑回测看收益", last: DomainAmbiguous, domain: DomainBacktestRun, mode: ModeExecute, playbook: true},
		{msg: "先问答，先不操作", last: DomainAmbiguous, domain: DomainChat, mode: ModeTalk},
		{msg: "帮我回测小米", last: DomainStockAnalysis, domain: DomainBacktestRun, mode: ModeExecute, playbook: true},
		{msg: "MACD", last: "", domain: DomainAmbiguous, mode: ModeClarify},
	}
	for _, tc := range cases {
		plan := p.Plan(PlanInput{UserText: tc.msg, LastDomain: tc.last})
		if plan.Domain != tc.domain || plan.Mode != tc.mode {
			t.Fatalf("%q last=%s: got %s/%s want %s/%s (%s)",
				tc.msg, tc.last, plan.Domain, plan.Mode, tc.domain, tc.mode, plan.Reason)
		}
		if plan.ShouldRunBacktestPlaybook() != tc.playbook {
			t.Fatalf("%q playbook=%v want %v", tc.msg, plan.ShouldRunBacktestPlaybook(), tc.playbook)
		}
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

func TestIntentPlannerUsesLLMOnlyInGrayZone(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"stock_analysis","confidence":0.8,"reason":"indicator mention"}`}
	p := IntentPlanner{Rules: RulePlanner{}, LLM: mock}

	hi := p.Plan(PlanInput{UserText: "腾讯现在怎么样"})
	if hi.Domain != DomainStockAnalysis {
		t.Fatalf("analysis domain=%s", hi.Domain)
	}
	if mock.calls != 0 {
		t.Fatalf("high-confidence analysis must not call LLM, calls=%d", mock.calls)
	}

	got := p.Plan(PlanInput{UserText: "MACD"})
	if mock.calls != 0 {
		t.Fatalf("ambiguous/clarify must not call classify LLM, calls=%d", mock.calls)
	}
	if got.Domain != DomainAmbiguous || got.Mode != ModeClarify {
		t.Fatalf("MACD should stay clarify, got %s/%s", got.Domain, got.Mode)
	}
}

func TestIntentPlannerRejectsBacktestWithoutVerb(t *testing.T) {
	mock := &classifyMock{body: `{"domain":"backtest_run","confidence":0.99,"reason":"signals"}`}
	p := IntentPlanner{Rules: RulePlanner{}, LLM: mock}
	got := p.Plan(PlanInput{UserText: "这个信号怎么样"})
	if got.ShouldRunBacktestPlaybook() || got.Domain == DomainBacktestRun {
		t.Fatalf("must not accept backtest_run without verb: %+v", got)
	}
	if got.Domain != DomainAmbiguous {
		t.Fatalf("keep clarify on blocked backtest, got %s", got.Domain)
	}
}

func TestIntentPlannerNilLLMKeepsRules(t *testing.T) {
	p := IntentPlanner{Rules: RulePlanner{}}
	got := p.Plan(PlanInput{UserText: "MACD"})
	if got.Domain != DomainAmbiguous {
		t.Fatalf("nil LLM should keep rules, got %s", got.Domain)
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
