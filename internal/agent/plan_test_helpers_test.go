package agent_test

import (
	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

func classifyFixture(extra map[string]string) *cognition.ClassifyFixtureProvider {
	base := map[string]string{
		"腾讯现在怎么样":                    cognition.FormatClassifyJSON("stock_analysis", "gather", "analyze", "test"),
		"换成贵州茅台":                     cognition.FormatClassifyJSON("stock_analysis", "gather", "symbol_resolve", "test"),
		"腾讯价格":                       cognition.FormatClassifyJSON("stock_analysis", "gather", "quote_price", "test"),
		"MACD":                         cognition.FormatClassifyJSON("ambiguous", "clarify", "", "test"),
		"帮我回测小米 SAR+MACD":             cognition.FormatClassifyJSON("backtest_run", "execute", "", "test"),
		"按知识库里的 4 小时 MACD 策略说明一下": cognition.FormatClassifyJSON("knowledge", "gather", "", "test"),
		"帮我查一下腾讯的股价":                 cognition.FormatClassifyJSON("stock_analysis", "gather", "quote_price", "test"),
	}
	for k, v := range extra {
		base[k] = v
	}
	return &cognition.ClassifyFixtureProvider{ByMessage: base}
}

func withClassifyPlanner(loop *agent.Loop, fixture *cognition.ClassifyFixtureProvider) {
	loop.SetPlanner(cognition.IntentPlanner{LLM: fixture})
}
