package skills

import "github.com/ghsemail/GeeGooAgent/internal/workflow"

// RegisterBuiltins registers all built-in skills into the given registry.
func RegisterBuiltins(r *Registry) {
	emptySteps := func() []workflow.Step { return []workflow.Step{} }
	r.Register(Spec{
		Name:         "premarket_market",
		Description:  "【市场盘前】CN/HK/US 宏观：指数 + 市场新闻 → market_premarket_report",
		PhaseA:       emptySteps,
		PerStock:     emptySteps,
		TemplatePath: "skills/premarket_market/template.md",
	})
	r.Register(Spec{
		Name:         "premarket_stock",
		Description:  "【个股盘前】读取市场盘前后，为 attitude 订阅标的逐股写 stock_premarket_report",
		PhaseA:       emptySteps,
		PerStock:     workflow.PerStockSteps,
		TemplatePath: "skills/premarket_stock/template.md",
	})
	r.Register(Spec{
		Name:         "intraday_stock",
		Description:  "盘中交易决策：持仓 + 盘前对照 + 小时级分析 + 现价，生成 intraday 报告",
		PhaseA:       workflow.IntradayPhaseASteps,
		PerStock:     workflow.IntradayPerStockSteps,
	})
	r.Register(Spec{
		Name:         "postmarket_stock",
		Description:  "盘后总结：小时级分析 + Bot 日志 + 盘前对照，生成 postmarket_stock 报告",
		PhaseA:       workflow.PostMarketPhaseASteps,
		PerStock:     workflow.PostMarketPerStockSteps,
	})
}
