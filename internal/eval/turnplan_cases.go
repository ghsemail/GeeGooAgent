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
			Description: "独立 session：查询腾讯控股现价。",
			Message:     "帮我查一下腾讯控股现在的股价",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_technical_chain", Title: "多轮 · 查价后看技术面",
			Description: "同 session：先查腾讯股价，再自然续问技术面/K 线。",
			SetupMessages: []string{"帮我查一下腾讯控股现在的股价"},
			Message:       "再帮我看看腾讯的技术面和K线图",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_symbol_switch", Title: "多轮 · 切换分析标的",
			Description: "同 session：先分析中际旭创，再明确改聊贵州茅台。",
			SetupMessages: []string{"帮我分析一下中际旭创"},
			Message:       "不聊中际旭创了，帮我分析一下贵州茅台",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_colloquial_ref", Title: "多轮 · 代词指代续问",
			Description: "同 session：建立中际旭创上下文后，用「它」续问同一标的走势。",
			SetupMessages: []string{"帮我分析一下中际旭创"},
			Message:       "它最近走势怎么样",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "signal_catalog_list", Title: "单轮 · 列信号策略",
			Description: "独立 session：列举可用信号/组合策略，plan 路由 dca_grid/gather 后由模型调 catalog 工具。",
			Message: "帮我看看我有哪些信号策略",
			ExpectDomain: "dca_grid", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_signal_combinations"},
		},
		// ── 信号探测 ──
		{
			ID: "signal_list_then_probe", Title: "多轮 · 列策略后测买卖点",
			Description: "同 session：先列出可用信号策略，再指定 SAR+MACD 测中际旭创买卖点。",
			SetupMessages: []string{"帮我看看我有哪些信号策略"},
			Message:       "我想用SAR加MACD组合，测一下中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		{
			ID: "signal_probe_direct", Title: "单轮 · 直接测买卖点",
			Description: "独立 session：显式指定标的与 signal_probe 意图。",
			Message: "帮我看看中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		// ── 策略回测 ──
		{
			ID: "backtest_explicit", Title: "单轮 · 显式回测",
			Description: "独立 session：指定 SAR+MACD 回测小米。",
			Message: "帮我用SAR加MACD回测一下小米",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "backtest_colloquial", Title: "单轮 · 口语回测",
			Description: "独立 session：省略策略名但仍应路由到 backtest_run。",
			Message: "帮我回测一下中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "analysis_then_backtest", Title: "多轮 · 分析后回测",
			Description: "同 session：先分析小米，再在同一语境下发起回测。",
			SetupMessages: []string{"帮我分析一下小米"},
			Message:       "接着用SAR加MACD帮小米跑个回测",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "strategy_list_then_backtest", Title: "多轮 · 选策略后回测",
			Description: "同 session：先问可回测策略，再指定 SAR+MACD 回测中际旭创。",
			SetupMessages: []string{"帮我看看有哪些可以回测的策略"},
			Message:       "用SAR加MACD组合回测中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"},
		},
		// ── 灰区 / 澄清 ──
		{
			ID: "ambiguous_bare_macd", Title: "单轮 · 模糊 MACD",
			Description: "独立 session：无上下文的 MACD 用法问题，应 clarify。",
			Message: "这个MACD信号平时该怎么用比较好",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		{
			ID: "compound_analysis_backtest", Title: "单轮 · 分析+回测复合",
			Description: "独立 session：一句话含分析与回测，应澄清而非直接执行。",
			Message: "帮我把中际旭创分析一下，然后再跑个回测看看效果",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest"},
		},
		// ── 闲聊 / QA ──
		{
			ID: "chat_definition", Title: "单轮 · 指标释义",
			Description: "独立 session：纯知识问答，plan 路由 chat/talk，不调用业务工具。",
			Message: "MACD 指标是什么意思",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"},
		},
		{
			ID: "chat_signal_quality", Title: "多轮 · 测点后问信号质量",
			Description: "同 session：先测买卖点，再问信号是否靠谱。",
			SetupMessages: []string{"帮我看看中际旭创有没有买卖点"},
			Message:       "刚才那个买卖点信号靠谱吗",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		// ── Bot 管理 ──
		{
			ID: "bot_reminder_list", Title: "单轮 · Reminder 列表",
			Description: "独立 session：查询 DCA reminder 列表。",
			Message: "我现在有哪些 Reminder 提醒",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_dca_reminders"},
		},
		{
			ID: "bot_grid_pnl", Title: "单轮 · 网格 Bot 盈亏",
			Description: "独立 session：查询腾讯网格 Bot 盈亏。",
			Message: "帮我看看腾讯网格 Bot 的盈亏情况",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_grid_bots"},
		},
		{
			ID: "bot_smarttrade_list", Title: "单轮 · SmartTrade 列表",
			Description: "独立 session：列出 SmartTrade 实例。",
			Message: "帮我查一下我运行中的 SmartTrade 有哪些",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_smart_trades"},
		},
		// ── 历史 / 报告 / 知识 / 新闻 ──
		{
			ID: "backtest_history", Title: "多轮 · 回测后查历史",
			Description: "同 session：先跑回测，再问小米上次回测结果。",
			SetupMessages: []string{"帮我用SAR加MACD回测一下小米"},
			Message:       "小米上次回测结果怎么样",
			ExpectDomain: "backtest_history", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_strategy_backtest_logs"},
		},
		{
			ID: "report_lookup", Title: "单轮 · 盘前报告",
			Description: "独立 session：查询盘前报告内容。",
			Message: "今天盘前报告写了什么内容",
			ExpectDomain: "report_lookup", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_stock_premarket_reports"},
		},
		{
			ID: "knowledge_lookup", Title: "单轮 · 知识库检索",
			Description: "独立 session：从知识库讲解 4H MACD。",
			Message: "按知识库帮我讲讲4小时MACD怎么用",
			ExpectDomain: "knowledge", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_knowledge"},
		},
		{
			ID: "news_lookup", Title: "单轮 · 新闻查询",
			Description: "独立 session：拉取市场新闻。",
			Message: "最近有什么财经新闻",
			ExpectDomain: "news", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"fetch_market_news"},
		},
		{
			ID: "dca_grid_backtest", Title: "单轮 · DCA 定投回测",
			Description: "独立 session：DCA/网格类回测意图。",
			Message: "帮我做一个DCA定投策略回测",
			ExpectDomain: "dca_grid", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"generate_dca_strategy"},
		},
	}
}

// defaultTurnPlanRuleTurns — plan-only 规则回归：用 LastDomain 模拟多轮上下文，不占 Chat session。
func defaultTurnPlanRuleTurns() []TurnPlanTurn {
	return []TurnPlanTurn{
		{ID: "stock_price", Message: "帮我查一下腾讯控股现在的股价",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code", "get_mcp_analysis"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "stock_technical_chain", Message: "再帮我看看腾讯的技术面和K线图", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code", "get_mcp_analysis"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "stock_symbol_switch", Message: "不聊中际旭创了，帮我分析一下贵州茅台", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "stock_colloquial_ref", Message: "它最近走势怎么样", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_code"}, ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "signal_catalog_list", Message: "帮我看看我有哪些信号策略",
			ExpectDomain: "dca_grid", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_signal_combinations"}},
		{ID: "signal_list_then_probe", Message: "我想用SAR加MACD组合，测一下中际旭创有没有买卖点", LastDomain: "dca_grid",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"probe_bot_signal_series"}},
		{ID: "signal_probe_direct", Message: "帮我看看中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"probe_bot_signal_series"}},
		{ID: "backtest_explicit", Message: "帮我用SAR加MACD回测一下小米",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "backtest_colloquial", Message: "帮我回测一下中际旭创",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "analysis_then_backtest", Message: "接着用SAR加MACD帮小米跑个回测", LastDomain: "stock_analysis",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "strategy_list_then_backtest", Message: "用SAR加MACD组合回测中际旭创", LastDomain: "dca_grid",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"run_strategy_backtest"}},
		{ID: "ambiguous_bare_macd", Message: "这个MACD信号平时该怎么用比较好",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"}},
		{ID: "compound_analysis_backtest", Message: "帮我把中际旭创分析一下，然后再跑个回测看看效果",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest"}},
		{ID: "chat_definition", Message: "MACD 指标是什么意思",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"}},
		{ID: "chat_signal_quality", Message: "刚才那个买卖点信号靠谱吗", LastDomain: "signal_probe",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"}},
		{ID: "bot_reminder_list", Message: "我现在有哪些 Reminder 提醒",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_dca_reminders"}},
		{ID: "bot_grid_pnl", Message: "帮我看看腾讯网格 Bot 的盈亏情况",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_grid_bots"}},
		{ID: "bot_smarttrade_list", Message: "帮我查一下我运行中的 SmartTrade 有哪些",
			ExpectDomain: "bot_manage", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_smart_trades"}},
		{ID: "backtest_history", Message: "小米上次回测结果怎么样", LastDomain: "backtest_run",
			ExpectDomain: "backtest_history", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"list_strategy_backtest_logs"}},
		{ID: "report_lookup", Message: "今天盘前报告写了什么内容",
			ExpectDomain: "report_lookup", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"get_stock_premarket_reports"}},
		{ID: "knowledge_lookup", Message: "按知识库帮我讲讲4小时MACD怎么用",
			ExpectDomain: "knowledge", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"search_knowledge"}},
		{ID: "news_lookup", Message: "最近有什么财经新闻",
			ExpectDomain: "news", ExpectMode: "gather", ExpectSOP: false,
			RequireTools: []string{"fetch_market_news"}},
		{ID: "dca_grid_backtest", Message: "帮我做一个DCA定投策略回测",
			ExpectDomain: "dca_grid", ExpectMode: "execute", ExpectSOP: false,
			RequireTools: []string{"generate_dca_strategy"}},
	}
}
