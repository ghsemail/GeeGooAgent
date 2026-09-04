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
			ID: "stock_analysis_explicit", Message: "腾讯现在怎么样",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code", "get_mcp_analysis"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "stock_colloquial", Message: "中际旭创呢",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"},
			ForbidTools:  []string{"run_strategy_backtest"},
		},
		{
			ID: "signal_probe", Message: "有没有买卖点",
			ExpectDomain: "signal_probe", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"probe_bot_signal_series"},
		},
		{
			ID: "backtest_explicit", Message: "帮我回测小米 SAR+MACD",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "ambiguous_bare_macd", Message: "MACD",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "probe_bot_signal_series"},
		},
		{
			ID: "compound_analysis_backtest", Message: "分析一下中际旭创再回测",
			ExpectDomain: "ambiguous", ExpectMode: "clarify", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest"},
		},
		{
			ID: "chat_definition", Message: "MACD 是什么",
			ExpectDomain: "chat", ExpectMode: "talk", ExpectSOP: false,
			ForbidTools: []string{"run_strategy_backtest", "get_mcp_analysis"},
		},
		{
			ID: "sticky_symbol_switch", Message: "换成贵州茅台", LastDomain: "stock_analysis",
			ExpectDomain: "stock_analysis", ExpectMode: "gather", ExpectSOP: true,
			RequireTools: []string{"search_code"},
		},
		{
			ID: "backtest_after_analysis", Message: "帮我回测小米", LastDomain: "stock_analysis",
			ExpectDomain: "backtest_run", ExpectMode: "execute", ExpectSOP: true,
			RequireTools: []string{"run_strategy_backtest"},
		},
	}
}
