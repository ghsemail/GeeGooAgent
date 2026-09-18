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
	r.Register(Spec{
		Name:        SkillStrategyDev,
		Description: "策略开发主 Workflow · Step1 策略认知：读策略库、LLM 合成 Agent 认知、写入/读回知识库",
		Chat: &ChatTrigger{
			Status:   "available",
			Triggers: []string{"策略认知", "策略开发", "学习策略", "了解策略", "写入知识库"},
			Phases: []string{
				"cognition_pick", "cognition_read_catalog",
				"cognition_compose", "cognition_save_kb", "cognition_verify_kb", "summarize",
			},
			ResumeHints: []string{"继续", "重试失败", "POST /v1/chat/workflow/resume"},
		},
	})
	r.Register(Spec{
		Name:        SkillMultiStrategyCompare,
		Description: "固定标的，串行 probe 多个策略并输出买/卖次对比表（Chat 关键词或 Scheduler cron）",
		Chat: &ChatTrigger{
			Status:   "available",
			Triggers: []string{"多策略", "对比", "挨个跑", "依次测", "哪个信号多"},
			Phases:   []string{"resolve_symbol", "pick_strategies", "probe_foreach", "summarize"},
			ResumeHints: []string{
				"继续", "重试失败", "POST /v1/chat/workflow/resume",
			},
		},
	})
	r.Register(Spec{
		Name:        SkillParamTune,
		Description: "策略参数调优（规划中）：串行尝试参数组合改善买卖点可见性",
		Chat: &ChatTrigger{
			Status:   "planned",
			Triggers: []string{"买卖点不明显", "信号太少", "帮我调参"},
			Phases: []string{
				"resolve_context", "baseline_probe", "pick_param_variants", "probe_foreach", "summarize",
			},
		},
	})
}
