package catalog

// HTTPSpec describes a GeeGooBot mcp-api forwarding tool.
type HTTPSpec struct {
	Name             string
	Description      string
	Path             string
	RequiresMCPToken bool
	DirectResponse   bool
	MergePayload     bool
	Parameters       map[string]any
}

// BespokeNames are implemented as dedicated handlers, not generic HTTP tools.
var BespokeNames = map[string]bool{
	"search_code": true, "web_search": true,
	"check_trading_day": true, "get_current_price": true, "get_report_bot_codes": true,
	"fetch_market_news": true, "fetch_stock_news": true, "get_mcp_analysis": true,
	"get_single_prompt_template": true,
	"get_stock_daily_reports": true, "list_today_reports": true, "list_today_post_market_reports": true, "get_capital_flow": true,
	"get_capital_distribution": true, "get_bot_yesterday_attitude": true,
	"recall_yesterday_summary": true, "read_working_state": true, "create_pre_market_report": true,
	"save_local_report": true, "write_execution_log": true, "recall": true,
}

// AllHTTP returns generic MCP HTTP tool specs (excludes bespoke names).
func AllHTTP() []HTTPSpec {
	raw := []HTTPSpec{
		{Name: "get_position", Description: "查询富途账户持仓。须先 search_code；无持仓或富途未配时 skip。", Path: "/getPosition", Parameters: codeQueryParameters("标的代码")},
		{Name: "get_ticker", Description: "盘中逐笔行情 (MCP /getTicker)；区别于 get_current_price 现价快照。需富途 OpenD；非交易时段可能 skip。", Path: "/getTicker", Parameters: codeQueryParameters("标的代码")},
		{Name: "get_broker", Description: "经纪席位分布。需富途；非交易时段或港股以外可能无数据。", Path: "/getBroker", Parameters: codeQueryParameters("标的代码")},
		{Name: "get_index_signals", Description: "列出 DCA 可用的单指标信号（SAR/MACD/BBAND 等）；每项含 signal_id、name、brief、info、frequency、index。用户未指定信号类型时，与 get_signal_combinations 二选一后向用户介绍并让其选定。", Path: "/getIndexSignalForSkill", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_signal_combinations", Description: "列出 DCA 可用的组合信号（buy_signal/sell_signal 指标链）；每项含 signal_id、name、brief、info。适合多指标共振；用户未指定时先问「单指标还是组合」，再展示 brief 供选择。", Path: "/getSignalCombinationForSkill", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_bot_log_by_type", Description: "按类型查询 Bot 日志。必填 type（DCA/GRID/SmartTrade/HDG 等）与 bot_id。", Path: "/getBotLogByType", Parameters: map[string]any{
			"type": "object", "required": []string{"type", "bot_id"},
			"properties": map[string]any{
				"type":   stringProp("Bot 类型"),
				"bot_id": stringProp("Bot _id"),
			},
		}},
		{Name: "generate_grid_strategy", Description: "生成 GRID 网格策略建议（LLM 分析 + 推荐上下限与网格数）。必填 code、name。返回 param 可直接作为 loopback_strategy 的 grid_param；回测前再调 loopback_strategy(type=grid)。", Path: "/generateGridStrategy", Parameters: generateGridStrategyParameters()},
		{Name: "generate_dca_strategy", Description: "生成 DCA 定投方案（趋势评估、信号适用性、动态/固定止盈止损）。必填 code、name、signal_id；signal_id 来自 get_index_signals 或 get_signal_combinations。返回 signal.buy_signal + dynamicParam/fixedParam 可组装 loopback_strategy(type=dca) 的 signal 与 sl_tp。", Path: "/generateDCAStrategy", Parameters: generateDCAStrategyParameters()},
		{Name: "loopback_strategy", Description: "策略历史回测（GeeGooSignal :3200）。勿裸调：grid 须先有 grid_param（generate_grid_strategy 的 param）；dca 须 signal（generate_dca_strategy 的 signal.buy_signal）与 sl_tp（按 comparison 选 dynamicParam 或 fixedParam 组装）。缺 fund/months_back 时先问用户。", Path: "/loopBackStrategy", DirectResponse: true, MergePayload: true, Parameters: loopbackStrategyParameters()},
		{Name: "create_competitor_prompt_template", Description: "创建竞品分析 Prompt 模板（高级）。chat 写操作需用户确认。", Path: "/createCompetitorPromptTemplate", MergePayload: true},
		{Name: "edit_competitor_prompt_template", Description: "编辑竞品分析 Prompt 模板。chat 写操作需用户确认。", Path: "/editCompetitorPromptTemplate", MergePayload: true},
		{Name: "delete_competitor_prompt_template", Description: "删除竞品分析 Prompt 模板。chat 写操作需用户确认。", Path: "/deleteCompetitorPromptTemplate", MergePayload: true},
		{Name: "create_etf_prompt_template", Description: "创建 ETF 分析 Prompt 模板（高级）。chat 写操作需用户确认。", Path: "/createEtfPromptTemplate", MergePayload: true},
		{Name: "edit_etf_prompt_template", Description: "编辑 ETF 分析 Prompt 模板。chat 写操作需用户确认。", Path: "/editEtfPromptTemplate", MergePayload: true},
		{Name: "delete_etf_prompt_template", Description: "删除 ETF 分析 Prompt 模板。chat 写操作需用户确认。", Path: "/deleteEtfPromptTemplate", MergePayload: true},
	}
	raw = append(raw, reportCRUD("pre_market", "盘前报告",
		"/createPreMarketReport", "/updatePreMarketReport", "/deletePreMarketReport", "/getPreMarketReports", false)...)
	raw = append(raw, reportCRUD("intraday", "盘中决策报告",
		"/createIntradayTradeDecisionReport", "/updateIntradayTradeDecisionReport", "/deleteIntradayTradeDecisionReport", "/getIntradayTradeDecisionReports", true)...)
	raw = append(raw, reportCRUD("post_market", "盘后报告",
		"/createPostMarketReport", "/updatePostMarketReport", "/deletePostMarketReport", "/getPostMarketReports", true)...)
	raw = append(raw, botCRUD("dca_bot", botKindDCA, "DCA 交易机器人", "/createDCABot", "/updateDCABot", "/deleteDCABot", "/getAllDCABots", "/getDCABotLog")...)
	raw = append(raw, botCRUD("grid_bot", botKindGrid, "GRID 网格交易机器人", "/createGRIDBot", "/updateGRIDBot", "/deleteGRIDBot", "/getAllGRIDBots", "/getGRIDBotLog")...)
	raw = append(raw, botCRUD("smart_trade", botKindSmartTrade, "SmartTrade 机器人", "/createSmartTrade", "/updateSmartTrade", "/deleteSmartTrade", "/getAllSmartTrades", "/getSmartTradeLog")...)
	raw = append(raw, botCRUD("hdg_bot", botKindHDG, "HDG 对冲机器人", "/createHDGBot", "/updateHDGBot", "/deleteHDGBot", "/getAllHDGBots", "/getHDGBotLog")...)
	raw = append(raw, botCRUD("dca_reminder", botKindDCAReminder, "DCA 提醒机器人", "/createDCAReminder", "/updateDCAReminder", "/deleteDCAReminder", "/getAllDCAReminders", "/getDCAReminderLog")...)
	raw = append(raw, botCRUD("grid_reminder", botKindGridReminder, "GRID 提醒机器人", "/createGRIDReminder", "/updateGRIDReminder", "/deleteGRIDReminder", "/getAllGRIDReminders", "/getGRIDReminderLog")...)
	raw = append(raw, botCRUD("smart_reminder", botKindSmartReminder, "Smart 提醒机器人", "/createSmartReminder", "/updateSmartReminder", "/deleteSmartReminder", "/getAllSmartReminders", "/getSmartReminderLog")...)

	out := make([]HTTPSpec, 0, len(raw))
	for _, spec := range raw {
		if BespokeNames[spec.Name] {
			continue
		}
		out = append(out, spec)
	}
	return out
}
