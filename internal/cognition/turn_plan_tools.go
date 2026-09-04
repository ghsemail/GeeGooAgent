package cognition

var (
	toolsChat = []string{}

	toolsStockAnalysis = []string{
		"search_code", "get_current_price", "get_ticker", "get_broker", "get_position",
		"check_trading_day", "get_mcp_analysis", "get_single_prompt_template",
		"get_single_prompt_template_by_index", "get_hourly_analysis_bundle",
		"get_capital_flow", "get_capital_distribution",
		"fetch_stock_news", "fetch_market_news", "web_search",
	}

	toolsNews = []string{
		"search_code", "fetch_stock_news", "fetch_market_news", "web_search",
	}

	toolsKnowledge = []string{"search_knowledge", "web_search"}

	toolsReportLookup = []string{
		"get_stock_premarket_reports", "get_stock_intraday_reports", "get_stock_postmarket_reports",
		"get_stock_daily_reports", "list_today_reports", "get_bot_yesterday_attitude",
	}

	toolsReportWrite = []string{
		"search_code",
		"create_stock_intraday_report", "update_stock_intraday_report", "delete_stock_intraday_report",
		"create_stock_postmarket_report", "update_stock_postmarket_report", "delete_stock_postmarket_report",
		"update_stock_premarket_report", "delete_stock_premarket_report",
		"get_stock_premarket_reports", "get_stock_intraday_reports", "get_stock_postmarket_reports",
	}

	toolsBotManage = []string{
		"search_code",
		"list_dca_bots", "create_dca_bot", "update_dca_bot", "delete_dca_bot", "get_dca_bot_log",
		"list_grid_bots", "create_grid_bot", "update_grid_bot", "delete_grid_bot", "get_grid_bot_log",
		"list_smart_trades", "create_smart_trade", "update_smart_trade", "delete_smart_trade", "get_smart_trade_log",
		"list_hdg_bots", "create_hdg_bot", "update_hdg_bot", "delete_hdg_bot", "get_hdg_bot_log",
		"list_dca_reminders", "create_dca_reminder", "update_dca_reminder", "delete_dca_reminder", "get_dca_reminder_log",
		"list_grid_reminders", "create_grid_reminder", "update_grid_reminder", "delete_grid_reminder", "get_grid_reminder_log",
		"list_smart_reminders", "create_smart_reminder", "update_smart_reminder", "delete_smart_reminder", "get_smart_reminder_log",
		"get_position", "get_bot_log_by_type",
	}

	toolsSignalProbe = []string{
		"search_code", "get_index_signals", "get_signal_combinations",
		"get_custom_signal_for_skill", "probe_bot_signal", "probe_bot_signal_series",
		"get_indicator_series",
	}

	toolsBacktestRun = []string{
		"search_code", "get_index_signals", "get_signal_combinations",
		"get_custom_signal_for_skill", "run_strategy_backtest", "get_indicator_series",
	}

	toolsBacktestHistory = []string{
		"list_strategy_backtest_logs", "get_strategy_backtest_log",
	}

	toolsCustomSignal = []string{
		"get_custom_signal", "get_custom_signal_for_skill", "get_all_custom_signal_id",
		"get_custom_strategy_definitions", "add_custom_signal", "edit_custom_signal", "delete_custom_signal",
	}

	toolsPromptAdmin = []string{
		"get_single_prompt_template", "get_single_prompt_template_by_index",
		"add_single_prompt_template", "edit_prompt_template", "delete_prompt_template", "switch_prompt_status",
		"create_competitor_prompt_template", "edit_competitor_prompt_template", "delete_competitor_prompt_template",
		"create_etf_prompt_template", "edit_etf_prompt_template", "delete_etf_prompt_template",
	}

	toolsDCAGrid = []string{
		"search_code", "get_index_signals", "get_signal_combinations",
		"generate_dca_strategy", "generate_grid_strategy", "loopback_strategy",
	}
)
