package cognition

import "testing"

func TestRecentSessionUtteranceRouting(t *testing.T) {
	p := IntentPlanner{Rules: RulePlanner{}}
	cases := []struct {
		msg    string
		last   Domain
		domain Domain
		mode   Mode
	}{
		{msg: "帮我查一下腾讯的股价", domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "可以，分析下技术面的价格和K线图", last: DomainStockAnalysis, domain: DomainStockAnalysis, mode: ModeGather},
		{msg: "帮我回测一下中际旭创", domain: DomainBacktestRun, mode: ModeExecute},
		{msg: "现在有哪些组合信号", domain: DomainAmbiguous, mode: ModeClarify},
		{msg: "这个信号准吗", domain: DomainChat, mode: ModeTalk},
		{msg: "我现在有哪些reminder", domain: DomainBotManage, mode: ModeGather},
		{msg: "帮我查看腾讯网格Bot的盈亏", domain: DomainBotManage, mode: ModeGather},
		{msg: "上次回测结果怎么样", domain: DomainBacktestHistory, mode: ModeGather},
		{msg: "今天盘前写了什么", domain: DomainReportLookup, mode: ModeGather},
		{msg: "按知识库讲 4H MACD", domain: DomainKnowledge, mode: ModeGather},
		{msg: "有什么新闻", domain: DomainNews, mode: ModeGather},
		{msg: "帮我做 dca 定投回测", domain: DomainDCAGrid, mode: ModeExecute},
		{msg: "就这个SAR加MACD组合信号，我想先测买卖点", domain: DomainSignalProbe, mode: ModeExecute},
		{msg: "帮我查一下我有哪些SmartTrade", domain: DomainBotManage, mode: ModeGather},
	}
	for _, tc := range cases {
		plan := p.Plan(PlanInput{UserText: tc.msg, LastDomain: tc.last})
		if plan.Domain != tc.domain || plan.Mode != tc.mode {
			t.Errorf("%q: got %s/%s want %s/%s (%s)", tc.msg, plan.Domain, plan.Mode, tc.domain, tc.mode, plan.Reason)
		}
	}
}
