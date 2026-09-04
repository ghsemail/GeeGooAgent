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
		{
			ID: "stock_analysis_explicit", Message: "帮我看看腾讯现在怎么样",
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
		{
			ID: "signal_probe", Message: "帮我看看中际旭创有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		{
			ID: "backtest_explicit", Message: "帮我用SAR加MACD回测一下小米",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
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
		{
			ID: "chat_definition", Message: "MACD 指标是什么意思",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"},
		},
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
