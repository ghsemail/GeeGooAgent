package catalog

// HTTPSpec describes a GeeGooBot mcp-api forwarding tool.
type HTTPSpec struct {
	Name             string
	Description      string
	Path             string
	RequiresMCPToken bool
	DirectResponse   bool
	MergePayload     bool
}

// BespokeNames are implemented as dedicated handlers, not generic HTTP tools.
var BespokeNames = map[string]bool{
	"check_trading_day": true, "get_current_price": true, "get_report_bot_codes": true,
	"fetch_market_news": true, "fetch_stock_news": true, "get_mcp_analysis": true,
	"get_stock_daily_reports": true, "list_today_reports": true, "get_capital_flow": true,
	"get_capital_distribution": true, "get_bot_yesterday_attitude": true,
	"recall_yesterday_summary": true, "read_working_state": true, "create_pre_market_report": true,
	"save_local_report": true, "send_feishu_summary": true, "write_execution_log": true, "recall": true,
}

func botCRUD(slug, label, create, update, delete, list, log string) []HTTPSpec {
	return []HTTPSpec{
		{Name: "create_" + slug, Description: "Create " + label + ".", Path: create, MergePayload: true},
		{Name: "update_" + slug, Description: "Update " + label + " by bot_id.", Path: update, MergePayload: true},
		{Name: "delete_" + slug, Description: "Delete " + label + " by bot_id.", Path: delete},
		{Name: "list_" + slug + "s", Description: "List all " + label + " bots.", Path: list},
		{Name: "get_" + slug + "_log", Description: "Get run log for " + label + ".", Path: log},
	}
}

func reportCRUD(slug, label, create, update, delete, list string, includeCreate bool) []HTTPSpec {
	var specs []HTTPSpec
	if includeCreate {
		specs = append(specs, HTTPSpec{
			Name: "create_" + slug + "_report", Description: "Create " + label + ".",
			Path: create, MergePayload: true,
		})
	}
	specs = append(specs,
		HTTPSpec{Name: "update_" + slug + "_report", Description: "Update " + label + " by report_id.", Path: update, MergePayload: true},
		HTTPSpec{Name: "delete_" + slug + "_report", Description: "Delete " + label + " by report_id.", Path: delete},
		HTTPSpec{Name: "get_" + slug + "_reports", Description: "Query stored " + label + " documents.", Path: list},
	)
	return specs
}

// AllHTTP returns generic MCP HTTP tool specs (excludes bespoke names).
func AllHTTP() []HTTPSpec {
	raw := []HTTPSpec{
		{Name: "search_code", Description: "Search stock by code or name.", Path: "/searchCode", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_position", Description: "Get account position for a symbol.", Path: "/getPosition"},
		{Name: "get_current_price", Description: "Get latest price.", Path: "/getCurrentPrice", DirectResponse: true},
		{Name: "get_ticker", Description: "盘中逐笔行情 (MCP /getTicker)；区别于 get_current_price 现价快照。", Path: "/getTicker"},
		{Name: "get_broker", Description: "Get broker distribution.", Path: "/getBroker"},
		{Name: "get_index_signals", Description: "List index signals.", Path: "/getIndexSignalForSkill", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_signal_combinations", Description: "List combined signals.", Path: "/getSignalCombinationForSkill", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_single_prompt_template", Description: "List prompt templates.", Path: "/getSinglePromptTemplate", DirectResponse: true},
		{Name: "get_bot_log_by_type", Description: "Query bot log by type.", Path: "/getBotLogByType"},
		{Name: "generate_grid_strategy", Description: "Generate GRID strategy.", Path: "/generateGridStrategy"},
		{Name: "generate_dca_strategy", Description: "Generate DCA strategy.", Path: "/generateDCAStrategy"},
		{Name: "loopback_strategy", Description: "Backtest strategy.", Path: "/loopBackStrategy", MergePayload: true},
		{Name: "create_competitor_prompt_template", Description: "Create competitor prompt template.", Path: "/createCompetitorPromptTemplate", MergePayload: true},
		{Name: "edit_competitor_prompt_template", Description: "Edit competitor prompt template.", Path: "/editCompetitorPromptTemplate", MergePayload: true},
		{Name: "delete_competitor_prompt_template", Description: "Delete competitor prompt template.", Path: "/deleteCompetitorPromptTemplate", MergePayload: true},
		{Name: "create_etf_prompt_template", Description: "Create ETF prompt template.", Path: "/createEtfPromptTemplate", MergePayload: true},
		{Name: "edit_etf_prompt_template", Description: "Edit ETF prompt template.", Path: "/editEtfPromptTemplate", MergePayload: true},
		{Name: "delete_etf_prompt_template", Description: "Delete ETF prompt template.", Path: "/deleteEtfPromptTemplate", MergePayload: true},
		{Name: "switch_bot", Description: "Enable or disable a bot/reminder.", Path: "/switchBot"},
	}
	raw = append(raw, reportCRUD("pre_market", "pre-market report",
		"/createPreMarketReport", "/updatePreMarketReport", "/deletePreMarketReport", "/getPreMarketReports", false)...)
	raw = append(raw, reportCRUD("intraday", "intraday trade decision report",
		"/createIntradayTradeDecisionReport", "/updateIntradayTradeDecisionReport", "/deleteIntradayTradeDecisionReport", "/getIntradayTradeDecisionReports", true)...)
	raw = append(raw, reportCRUD("post_market", "post-market report",
		"/createPostMarketReport", "/updatePostMarketReport", "/deletePostMarketReport", "/getPostMarketReports", true)...)
	raw = append(raw, botCRUD("dca_bot", "DCA trading bot", "/createDCABot", "/updateDCABot", "/deleteDCABot", "/getAllDCABots", "/getDCABotLog")...)
	raw = append(raw, botCRUD("grid_bot", "GRID trading bot", "/createGRIDBot", "/updateGRIDBot", "/deleteGRIDBot", "/getAllGRIDBots", "/getGRIDBotLog")...)
	raw = append(raw, botCRUD("smart_trade", "SmartTrade bot", "/createSmartTrade", "/updateSmartTrade", "/deleteSmartTrade", "/getAllSmartTrades", "/getSmartTradeLog")...)
	raw = append(raw, botCRUD("hdg_bot", "HDG hedging bot", "/createHDGBot", "/updateHDGBot", "/deleteHDGBot", "/getAllHDGBots", "/getHDGBotLog")...)
	raw = append(raw, botCRUD("dca_reminder", "DCA reminder", "/createDCAReminder", "/updateDCAReminder", "/deleteDCAReminder", "/getAllDCAReminders", "/getDCAReminderLog")...)
	raw = append(raw, botCRUD("grid_reminder", "GRID reminder", "/createGRIDReminder", "/updateGRIDReminder", "/deleteGRIDReminder", "/getAllGRIDReminders", "/getGRIDReminderLog")...)
	raw = append(raw, botCRUD("smart_reminder", "Smart reminder", "/createSmartReminder", "/updateSmartReminder", "/deleteSmartReminder", "/getAllSmartReminders", "/getSmartReminderLog")...)

	out := make([]HTTPSpec, 0, len(raw))
	for _, spec := range raw {
		if BespokeNames[spec.Name] {
			continue
		}
		out = append(out, spec)
	}
	return out
}
