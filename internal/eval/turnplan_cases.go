package eval

// TurnPlanTurn is one user utterance and the expected routing decision (plan-only / rule regression).
type TurnPlanTurn struct {
	ID           string   `json:"id"`
	Message      string   `json:"message"`
	LastDomain   string   `json:"last_domain,omitempty"`
	ExpectDomain string   `json:"expect_domain"`
	ExpectMode   string   `json:"expect_mode"`
	ExpectSOP    bool     `json:"expect_sop"`
	ForbidTools  []string `json:"forbid_tools,omitempty"`
	RequireTools []string `json:"require_tools,omitempty"`
}

// TurnPlanLiveCase is one independent Dock Chat eval session (setup + final utterance).
type TurnPlanLiveCase struct {
	ID            string
	Title         string
	Description   string
	SetupMessages []string
	Message       string
	ExpectDomain  string
	ExpectMode    string
	ExpectSOP     bool
	ForbidTools   []string
	RequireTools  []string
}

// TurnPlanSuite is the options_json shape for dashboard eval case category=turn_plan (plan_only bundle).
type TurnPlanSuite struct {
	Category       string         `json:"category"`
	PlanOnly       bool           `json:"plan_only"`
	SessionCleanup string         `json:"session_cleanup"`
	DualModelEval  bool           `json:"dual_model_eval"`
	Turns          []TurnPlanTurn `json:"turns"`
}

// DefaultTurnPlanSuite is the canonical routing regression suite (plan-only API, no Chat session).
func DefaultTurnPlanSuite() TurnPlanSuite {
	return TurnPlanSuite{
		Category:       "turn_plan",
		PlanOnly:       true,
		SessionCleanup: "before_run",
		DualModelEval:  false,
		Turns:          defaultTurnPlanRuleTurns(),
	}
}

// defaultTurnPlanLiveCases — 每条用例 = 一个新 session；多轮对话写在 SetupMessages 里按序发送。
func defaultTurnPlanLiveCases() []TurnPlanLiveCase {
	return []TurnPlanLiveCase{
		// ── 股票分析 ──
		{
			ID: "stock_price", Title: "单轮 · 查股价",
			Description: "独立 session：查询腾讯控股现价，走 stock_analysis gather SOP。",
			Message:     "帮我查一下腾讯的股价",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_technical_chain", Title: "多轮 · 查价后看技术面",
			Description: "同 session：先查腾讯股价，再跟进技术面/K 线分析。",
			SetupMessages: []string{"帮我查一下腾讯的股价"},
			Message:       "可以，分析下技术面的价格和K线图",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_symbol_switch", Title: "多轮 · 切换分析标的",
			Description: "同 session：先分析中际旭创，再口语切换到贵州茅台。",
			SetupMessages: []string{"帮我分析一下中际旭创"},
			Message:       "那就换成贵州茅台吧",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_colloquial_ref", Title: "多轮 · 口语指代续问",
			Description: "同 session：建立中际旭创上下文后，用「这边呢」续问同一标的。",
			SetupMessages: []string{"帮我分析一下中际旭创"},
			Message:       "中际旭创这边呢",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		// ── 信号探测 ──
		{
			ID: "signal_list_then_probe", Title: "多轮 · 列策略后测买卖点",
			Description: "同 session：先列出可用信号策略，再指定 SAR+MACD 测中际旭创买卖点。",
			SetupMessages: []string{"帮我看看我有哪些信号策略"},
			Message:       "就用SAR加MACD组合，帮我测一下中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		{
			ID: "signal_probe_direct", Title: "单轮 · 直接测买卖点",
			Description: "独立 session：显式指定标的与 signal_probe 意图。",
			Message: "帮我看看中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		// ── 策略回测 ──
		{
			ID: "backtest_explicit", Title: "单轮 · 显式回测",
			Description: "独立 session：指定 SAR+MACD 回测小米。",
			Message: "帮我用SAR加MACD回测一下小米",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "backtest_colloquial", Title: "单轮 · 口语回测",
			Description: "独立 session：省略策略名但仍应路由到 backtest_run。",
			Message: "帮我回测一下中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "analysis_then_backtest", Title: "多轮 · 分析后回测",
			Description: "同 session：先分析小米，再在同一语境下发起回测。",
			SetupMessages: []string{"帮我分析一下小米"},
			Message:       "好，那帮我用SAR加MACD回测一下",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "strategy_list_then_backtest", Title: "多轮 · 选策略后回测",
			Description: "同 session：先问可回测策略，再选 SAR+MACD 回测中际旭创。",
			SetupMessages: []string{"我有哪些可以回测的策略"},
			Message:       "选SAR加MACD，回测中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		// ── 灰区 / 澄清 ──
		{
			ID: "ambiguous_bare_macd", Title: "单轮 · 模糊 MACD",
			Description: "独立 session：无上下文的「这个 MACD 信号怎么弄」，应 clarify 而非直接执行。",
			Message: "这个MACD信号怎么弄比较好",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		{
			ID: "compound_analysis_backtest", Title: "单轮 · 分析+回测复合",
			Description: "独立 session：一句话含分析与回测，应澄清而非直接执行。",
			Message: "帮我把中际旭创分析一下，再跑个回测看看",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest"},
		},
		// ── 闲聊 / QA ──
		{
			ID: "chat_definition", Title: "单轮 · 指标释义",
			Description: "独立 session：纯知识问答，不触发分析 SOP。",
			Message: "MACD 指标是什么意思",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"},
		},
		{
			ID: "chat_signal_quality", Title: "多轮 · 测点后问信号质量",
			Description: "同 session：先 probe 买卖点，再问「这个信号准吗」走 talk。",
			SetupMessages: []string{"帮我看看中际旭创有没有买卖点"},
			Message:       "这个信号准吗",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		// ── Bot 管理 ──
		{
			ID: "bot_reminder_list", Title: "单轮 · Reminder 列表",
			Description: "独立 session：查询 DCA reminder 列表。",
			Message: "我现在有哪些reminder",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_dca_reminders"},
		},
		{
			ID: "bot_grid_pnl", Title: "单轮 · 网格 Bot 盈亏",
			Description: "独立 session：查询腾讯网格 Bot 盈亏。",
			Message: "帮我查看腾讯网格Bot的盈亏",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_grid_bots"},
		},
		{
			ID: "bot_smarttrade_list", Title: "单轮 · SmartTrade 列表",
			Description: "独立 session：列出 SmartTrade 实例。",
			Message: "帮我查一下我有哪些SmartTrade",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_smart_trades"},
		},
		// ── 历史 / 报告 / 知识 / 新闻 ──
		{
			ID: "backtest_history", Title: "多轮 · 回测后查历史",
			Description: "同 session：先跑回测，再问「上次回测结果怎么样」。",
			SetupMessages: []string{"帮我用SAR加MACD回测一下小米"},
			Message:       "上次回测结果怎么样",
			ExpectDomain: "backtest_history", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_strategy_backtest_logs"},
		},
		{
			ID: "report_lookup", Title: "单轮 · 盘前报告",
			Description: "独立 session：查询盘前报告内容。",
			Message: "今天盘前写了什么",
			ExpectDomain: "report_lookup", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_stock_premarket_reports"},
		},
		{
			ID: "knowledge_lookup", Title: "单轮 · 知识库检索",
			Description: "独立 session：从知识库讲解 4H MACD。",
			Message: "按知识库讲 4H MACD",
			ExpectDomain: "knowledge", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_knowledge"},
		},
		{
			ID: "news_lookup", Title: "单轮 · 新闻查询",
			Description: "独立 session：拉取市场新闻。",
			Message: "有什么新闻",
			ExpectDomain: "news", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"fetch_market_news"},
		},
		{
			ID: "dca_grid_backtest", Title: "单轮 · DCA 定投回测",
			Description: "独立 session：DCA/网格类回测意图。",
			Message: "帮我做 dca 定投回测",
			ExpectDomain: "dca_grid", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"generate_dca_strategy"},
		},
	}
}

// defaultTurnPlanRuleTurns — plan-only 规则回归：用 LastDomain 模拟多轮上下文，不占 Chat session。
func defaultTurnPlanRuleTurns() []TurnPlanTurn {
	return []TurnPlanTurn{
		{ID: "stock_price", Message: "帮我查一下腾讯的股价",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "stock_technical_chain", Message: "可以，分析下技术面的价格和K线图", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "stock_symbol_switch", Message: "那就换成贵州茅台吧", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "stock_colloquial_ref", Message: "中际旭创这边呢", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "signal_list_then_probe", Message: "就用SAR加MACD组合，帮我测一下中际旭创有没有买卖点", LastDomain: "signal_probe",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"}},
		{ID: "signal_probe_direct", Message: "帮我看看中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"}},
		{ID: "backtest_explicit", Message: "帮我用SAR加MACD回测一下小米",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "backtest_colloquial", Message: "帮我回测一下中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "analysis_then_backtest", Message: "好，那帮我用SAR加MACD回测一下", LastDomain: "stock_analysis",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "strategy_list_then_backtest", Message: "选SAR加MACD，回测中际旭创", LastDomain: "backtest_run",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "ambiguous_bare_macd", Message: "这个MACD信号怎么弄比较好",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"}},
		{ID: "compound_analysis_backtest", Message: "帮我把中际旭创分析一下，再跑个回测看看",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "chat_definition", Message: "MACD 指标是什么意思",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"}},
		{ID: "chat_signal_quality", Message: "这个信号准吗", LastDomain: "signal_probe",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"}},
		{ID: "bot_reminder_list", Message: "我现在有哪些reminder",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_dca_reminders"}},
		{ID: "bot_grid_pnl", Message: "帮我查看腾讯网格Bot的盈亏",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_grid_bots"}},
		{ID: "bot_smarttrade_list", Message: "帮我查一下我有哪些SmartTrade",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_smart_trades"}},
		{ID: "backtest_history", Message: "上次回测结果怎么样", LastDomain: "backtest_run",
			ExpectDomain: "backtest_history", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_strategy_backtest_logs"}},
		{ID: "report_lookup", Message: "今天盘前写了什么",
			ExpectDomain: "report_lookup", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_stock_premarket_reports"}},
		{ID: "knowledge_lookup", Message: "按知识库讲 4H MACD",
			ExpectDomain: "knowledge", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_knowledge"}},
		{ID: "news_lookup", Message: "有什么新闻",
			ExpectDomain: "news", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"fetch_market_news"}},
		{ID: "dca_grid_backtest", Message: "帮我做 dca 定投回测",
			ExpectDomain: "dca_grid", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"generate_dca_strategy"}},
	}
}
