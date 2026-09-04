package eval

// TurnPlanTurn is one user utterance and the expected routing decision.
type TurnPlanTurn struct {
	ID            string   `json:"id"`
	Message       string   `json:"message"`
	LastDomain    string   `json:"last_domain,omitempty"`
	ExpectDomain  string   `json:"expect_domain"`
	ExpectMode    string   `json:"expect_mode"`
	ExpectSOP     bool     `json:"expect_sop"`
	ForbidTools   []string `json:"forbid_tools,omitempty"`
	RequireTools  []string `json:"require_tools,omitempty"`
}

// TurnPlanSuite is the options_json shape for dashboard eval case category=turn_plan.
type TurnPlanSuite struct {
	Category        string         `json:"category"`
	PlanOnly        bool           `json:"plan_only"`
	SessionCleanup  string         `json:"session_cleanup"`
	DualModelEval   bool           `json:"dual_model_eval"`
	Turns           []TurnPlanTurn `json:"turns"`
}

// DefaultTurnPlanSuite is the canonical routing regression suite (SSOT for SQL seeds).
func DefaultTurnPlanSuite() TurnPlanSuite {
	return TurnPlanSuite{
		Category:       "turn_plan",
		PlanOnly:         true,
		SessionCleanup:   "before_run",
		DualModelEval:    false,
		Turns:            defaultTurnPlanTurns(),
	}
}

func defaultTurnPlanTurns() []TurnPlanTurn {
	return []TurnPlanTurn{
		// stock-analysis: session 真实多轮
		{
			ID: "stock_price_lookup", Message: "帮我查一下腾讯的股价",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_followup_technical", Message: "可以，分析下技术面的价格和K线图", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_colloquial", Message: "中际旭创这边呢",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		// strategy-signal-probe
		{
			ID: "signal_probe", Message: "帮我看看中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		{
			ID: "signal_probe_combo", Message: "就这个SAR加MACD组合信号，我想先测买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		// strategy-backtest-run
		{
			ID: "backtest_explicit", Message: "帮我用SAR加MACD回测一下小米",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "backtest_colloquial", Message: "帮我回测一下中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		// gray-zone / compound
		{
			ID: "ambiguous_bare_macd", Message: "这个MACD信号怎么弄比较好",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		{
			ID: "compound_analysis_backtest", Message: "帮我把中际旭创分析一下，再跑个回测看看",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest"},
		},
		// chat / QA
		{
			ID: "chat_definition", Message: "MACD 指标是什么意思",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"},
		},
		{
			ID: "chat_signal_quality", Message: "这个信号准吗",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		// bot-manager: session 真实话术
		{
			ID: "bot_reminder_list", Message: "我现在有哪些reminder",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_dca_reminders"},
		},
		{
			ID: "bot_grid_pnl", Message: "帮我查看腾讯网格Bot的盈亏",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_grid_bots"},
		},
		{
			ID: "bot_smarttrade_list", Message: "帮我查一下我有哪些SmartTrade",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_smart_trades"},
		},
		// strategy-backtest-history
		{
			ID: "backtest_history", Message: "上次回测结果怎么样",
			ExpectDomain: "backtest_history", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_strategy_backtest_logs"},
		},
		// report-lookup
		{
			ID: "report_lookup", Message: "今天盘前写了什么",
			ExpectDomain: "report_lookup", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_stock_premarket_reports"},
		},
		// knowledge-base
		{
			ID: "knowledge_lookup", Message: "按知识库讲 4H MACD",
			ExpectDomain: "knowledge", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_knowledge"},
		},
		// news
		{
			ID: "news_lookup", Message: "有什么新闻",
			ExpectDomain: "news", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"fetch_market_news"},
		},
		// strategy-backtest (DCA/网格)
		{
			ID: "dca_grid_backtest", Message: "帮我做 dca 定投回测",
			ExpectDomain: "dca_grid", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"generate_dca_strategy"},
		},
		// sticky multi-turn
		{
			ID: "sticky_symbol_switch", Message: "那就换成贵州茅台吧", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"},
		},
		{
			ID: "backtest_after_analysis", Message: "好，那帮我回测一下小米", LastDomain: "stock_analysis",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
	}
}
