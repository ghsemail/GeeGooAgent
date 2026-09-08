package cognition

import "testing"

func TestRecentSessionUtteranceRouting(t *testing.T) {
	fixture := &ClassifyFixtureProvider{ByMessage: map[string]string{
		"帮我查一下腾讯的股价":              FormatClassifyJSON("stock_analysis", "gather", "quote_price", "fixture"),
		"再帮我看看腾讯的技术面和K线图":         FormatClassifyJSON("stock_analysis", "gather", "technical_analysis", "fixture"),
		"不聊中际旭创了，帮我分析一下贵州茅台":     FormatClassifyJSON("stock_analysis", "gather", "symbol_resolve", "fixture"),
		"它最近走势怎么样":                FormatClassifyJSON("stock_analysis", "gather", "context_followup", "fixture"),
		"帮我回测一下中际旭创":              FormatClassifyJSON("backtest_run", "execute", "", "fixture"),
		"现在有哪些组合信号":               FormatClassifyJSON("dca_grid", "gather", "", "fixture"),
		"帮我看看我有哪些信号策略":            FormatClassifyJSON("dca_grid", "gather", "", "fixture"),
		"刚才那个买卖点信号靠谱吗":            FormatClassifyJSON("chat", "talk", "", "fixture"),
		"我现在有哪些 Reminder 提醒":      FormatClassifyJSON("bot_manage", "gather", "", "fixture"),
		"帮我看看腾讯网格 Bot 的盈亏情况":      FormatClassifyJSON("bot_manage", "gather", "", "fixture"),
		"小米上次回测结果怎么样":             FormatClassifyJSON("backtest_history", "gather", "", "fixture"),
		"今天盘前报告写了什么内容":            FormatClassifyJSON("report_lookup", "gather", "", "fixture"),
		"按知识库帮我讲讲4小时MACD怎么用":      FormatClassifyJSON("knowledge", "gather", "", "fixture"),
		"最近有什么财经新闻":               FormatClassifyJSON("news", "gather", "", "fixture"),
		"帮我做一个DCA定投策略回测":          FormatClassifyJSON("dca_grid", "execute", "", "fixture"),
		"就这个SAR加MACD组合信号，我想先测买卖点": FormatClassifyJSON("signal_probe", "execute", "", "fixture"),
		"帮我查一下我运行中的 SmartTrade 有哪些": FormatClassifyJSON("bot_manage", "gather", "", "fixture"),
	}}
	p := IntentPlanner{LLM: fixture}
	cases := []struct {
		msg    string
		last   Domain
		domain Domain
		mode   Mode
	}{
		{msg: "帮我查一下腾讯的股价", domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "再帮我看看腾讯的技术面和K线图", last: DomainStockAnalysis, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "不聊中际旭创了，帮我分析一下贵州茅台", last: DomainStockAnalysis, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "它最近走势怎么样", last: DomainStockAnalysis, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "帮我回测一下中际旭创", domain: DomainBacktestRun, mode: ModeExecute},
		{msg: "现在有哪些组合信号", domain: DomainDCAGrid, mode: ModeGather},
		{msg: "帮我看看我有哪些信号策略", domain: DomainDCAGrid, mode: ModeGather},
		{msg: "刚才那个买卖点信号靠谱吗", domain: DomainChat, mode: ModeTalk},
		{msg: "我现在有哪些 Reminder 提醒", domain: DomainBotManage, mode: ModeGather},
		{msg: "帮我看看腾讯网格 Bot 的盈亏情况", domain: DomainBotManage, mode: ModeGather},
		{msg: "小米上次回测结果怎么样", domain: DomainBacktestHistory, mode: ModeGather},
		{msg: "今天盘前报告写了什么内容", domain: DomainReportLookup, mode: ModeGather},
		{msg: "按知识库帮我讲讲4小时MACD怎么用", domain: DomainKnowledge, mode: ModeGather},
		{msg: "最近有什么财经新闻", domain: DomainNews, mode: ModeGather},
		{msg: "帮我做一个DCA定投策略回测", domain: DomainDCAGrid, mode: ModeExecute},
		{msg: "就这个SAR加MACD组合信号，我想先测买卖点", domain: DomainSignalProbe, mode: ModeExecute},
		{msg: "帮我查一下我运行中的 SmartTrade 有哪些", domain: DomainBotManage, mode: ModeGather},
	}
	for _, tc := range cases {
		plan := p.Plan(PlanInput{UserText: tc.msg, LastDomain: tc.last})
		if plan.Domain != tc.domain || plan.Mode != tc.mode {
			t.Errorf("%q: got %s/%s want %s/%s (%s)", tc.msg, plan.Domain, plan.Mode, tc.domain, tc.mode, plan.Reason)
		}
	}
}
