package skills

import "github.com/ghsemail/GeeGooAgent/internal/workflow"

// RegisterBuiltins registers all built-in skills into the given registry.
func RegisterBuiltins(r *Registry) {
	emptySteps := func() []workflow.Step { return []workflow.Step{} }
	r.Register(Spec{
		Name:         "pre_market",
		Description:  "市场盘前报告：按 CN/HK/US 生成全局宏观盘前（指数 + 市场新闻）",
		PhaseA:       emptySteps,
		PerStock:     emptySteps,
		TemplatePath: "skills/pre_market/template.md",
		ManifestPath: "skills/pre_market/manifest.yaml",
	})
	r.Register(Spec{
		Name:         "pre_market_stock",
		Description:  "股票盘前报告：引用市场报告，为 attitude 订阅标的逐股生成报告（保留 bot_id 绑定）",
		PhaseA:       emptySteps,
		PerStock:     workflow.PerStockSteps,
		TemplatePath: "skills/pre_market_stock/template.md",
		ManifestPath: "skills/pre_market_stock/manifest.yaml",
	})
	r.Register(Spec{
		Name:         "intraday",
		Description:  "盘中交易决策：持仓 + 盘前对照 + 小时级分析 + 现价，生成 intraday 报告",
		PhaseA:       workflow.IntradayPhaseASteps,
		PerStock:     workflow.IntradayPerStockSteps,
		TemplatePath: "skills/intraday/template.md",
		ManifestPath: "skills/intraday/manifest.yaml",
	})
	r.Register(Spec{
		Name:         "post_market",
		Description:  "盘后总结：小时级分析 + Bot 日志 + 盘前对照，生成 post_market 报告",
		PhaseA:       workflow.PostMarketPhaseASteps,
		PerStock:     workflow.PostMarketPerStockSteps,
		TemplatePath: "skills/post_market/template.md",
		ManifestPath: "skills/post_market/manifest.yaml",
	})
}
