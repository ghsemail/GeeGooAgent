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
	"get_hourly_analysis_bundle": true,
	"get_single_prompt_template": true,
	"get_stock_daily_reports": true, "list_today_reports": true, "list_today_stock_postmarket_reports": true, "get_capital_flow": true,
	"get_capital_distribution": true, "get_bot_yesterday_attitude": true,
	"recall_yesterday_summary": true, "read_working_state": true, "create_stock_premarket_report": true,
	"create_market_premarket_report": true, "get_market_premarket_report": true,
	"save_local_report": true, "write_execution_log": true, "recall": true,
}

// AllHTTP returns generic MCP HTTP tool specs (excludes bespoke names).
func AllHTTP() []HTTPSpec {
	raw := []HTTPSpec{
		{Name: "get_position", Description: "查询富途账户持仓。须先 search_code；无持仓或富途未配时 skip。", Path: "/getPosition", Parameters: codeQueryParameters("标的代码")},
		{Name: "get_ticker", Description: "盘中逐笔行情 (MCP /getTicker)；区别于 get_current_price 现价快照。需富途 OpenD；非交易时段可能 skip。", Path: "/getTicker", Parameters: codeQueryParameters("标的代码")},
		{Name: "get_broker", Description: "经纪席位分布。需富途；非交易时段或港股以外可能无数据。", Path: "/getBroker", Parameters: codeQueryParameters("标的代码")},
		{Name: "get_index_signals", Description: "列出 DCA 可用的单指标信号（SAR/MACD/BBAND 等）；每项含 signal_id、name、brief、info、frequency、index。用户未指定信号类型时，与 get_signal_combinations 二选一后向用户介绍并让其选定。", Path: "/getIndexSignalForSkill", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_signal_combinations", Description: "列出 DCA 组合信号摘要（signal_id、name、brief、frequency、buy/sell 规则数与 indexes）。用户选定 signal_id 后，用该项完整 buy_signal/sell_signal 调 probe。适合多指标共振；未指定时展示 brief 供选择。", Path: "/getSignalCombinationForSkill", RequiresMCPToken: false, DirectResponse: true},
		{Name: "get_bot_log_by_type", Description: "按类型查询 Bot 日志。必填 type（DCA/GRID/SmartTrade/HDG 等）与 bot_id。", Path: "/getBotLogByType", Parameters: map[string]any{
			"type": "object", "required": []string{"type", "bot_id"},
			"properties": map[string]any{
				"type":   stringProp("Bot 类型"),
				"bot_id": stringProp("Bot _id"),
			},
		}},
		{Name: "generate_grid_strategy", Description: "生成 GRID 网格策略建议（LLM 分析 + JSON 批量英译）。必填 code、name。cn 约 40s、en 约 50s。返回 param 可直接作为 loopback_strategy 的 grid_param；回测前再调 loopback_strategy(type=grid)。", Path: "/generateGridStrategy", Parameters: generateGridStrategyParameters()},
		{Name: "generate_dca_strategy", Description: "生成 DCA 定投方案（趋势评估、信号适用性、动态/固定止盈止损；JSON 批量英译）。必填 code、name、signal_id（推荐 get_signal_combinations 组合信号）。cn 约 2～2.5min。返回 signal.buy_signal + dynamicParam/fixedParam 可组装 loopback_strategy(type=dca) 的 signal 与 sl_tp。", Path: "/generateDCAStrategy", Parameters: generateDCAStrategyParameters()},
		{Name: "loopback_strategy", Description: "策略历史回测（GeeGooSignal :3200）。勿裸调：grid 须先有 grid_param（generate_grid_strategy 的 param）；dca 须 signal（generate_dca_strategy 的 signal.buy_signal）与 sl_tp（按 comparison 选 dynamicParam 或 fixedParam 组装）。缺 fund/months_back 时先问用户。", Path: "/loopBackStrategy", DirectResponse: true, MergePayload: true, Parameters: loopbackStrategyParameters()},
		{Name: "probe_bot_signal_series", Description: "策略开发/回测信号探测（时间序列）。返回 K 线、buy_merged/sell_merged 及每条规则的 signal_series、value_series、reasons。与 trading_operation「策略开发-信号测试」「回测运行」第一步相同。定制策略 buy_signal 用 custom.index。", Path: "/probeBotSignalSeries", DirectResponse: true, MergePayload: true, Parameters: probeBotSignalSeriesParameters()},
		{Name: "run_strategy_backtest", Description: "SmartTrade/catalog 信号策略回测（probe+模拟+落库，与 trading_operation「回测运行」一致）。用户要「跑回测/看收益回撤/成交笔数」且未明确 DCA/网格/定投时**优先调用**；返回 log_id、profit_rate、final_value、trade_count。必填 code、frequency、buy_signal；可选 sell_signal、strategy_label、period(1m/3m)、fund(默认100000)、months_back、trade_config。勿用于 DCA/Grid——那走 generate_* + loopback_strategy。", Path: "/runStrategyBacktest", DirectResponse: true, MergePayload: true, Parameters: runStrategyBacktestParameters()},
		{Name: "probe_bot_signal", Description: "单 bar 信号探测（默认最后一根 K 线）。返回买卖信号、各规则指标值与 reason。适合快速验证当前 bar 是否触发。", Path: "/probeBotSignal", DirectResponse: true, MergePayload: true, Parameters: probeBotSignalParameters()},
		{Name: "get_indicator_series", Description: "拉取指标数值序列，用于动态止盈/止损线（role=sl 或 tp）。回测前配合 probe_bot_signal_series 使用。", Path: "/getIndicatorSeries", DirectResponse: true, MergePayload: true, Parameters: getIndicatorSeriesParameters()},
		{Name: "list_strategy_backtest_logs", Description: "列出策略回测历史摘要（不含 chart_data）。用户要「最新/有哪些回测记录」时优先调用；返回 log_id、code、strategy_label、created_at、result.profit/profit_rate、trade_count、has_chart_data。需要 K 线或成交时间线时再 get_strategy_backtest_log(log_id)。", Path: "/listStrategyBacktestLogs", DirectResponse: true, MergePayload: true, Parameters: listStrategyBacktestLogsParameters()},
		{Name: "get_strategy_backtest_log", Description: "读取单条策略回测详情：run（含 result、chart_data.probe、snapshots）与 trades 时间线。用于分析最新/指定回测结果。", Path: "/getStrategyBacktestLog", DirectResponse: true, MergePayload: true, Parameters: getStrategyBacktestLogParameters()},
		{Name: "create_competitor_prompt_template", Description: "创建竞品分析用户 Prompt（analyze 服务，非 single_prompt_template 列表）。写操作需用户确认。", Path: "/createCompetitorPromptTemplate", MergePayload: true},
		{Name: "edit_competitor_prompt_template", Description: "编辑竞品分析用户 Prompt。写操作需用户确认。", Path: "/editCompetitorPromptTemplate", MergePayload: true},
		{Name: "delete_competitor_prompt_template", Description: "删除竞品分析用户 Prompt。写操作需用户确认。", Path: "/deleteCompetitorPromptTemplate", MergePayload: true},
		{Name: "create_etf_prompt_template", Description: "创建 ETF 分析用户 Prompt（analyze 服务）。写操作需用户确认。", Path: "/createEtfPromptTemplate", MergePayload: true},
		{Name: "edit_etf_prompt_template", Description: "编辑 ETF 分析用户 Prompt。写操作需用户确认。", Path: "/editEtfPromptTemplate", MergePayload: true},
		{Name: "delete_etf_prompt_template", Description: "删除 ETF 分析用户 Prompt。写操作需用户确认。", Path: "/deleteEtfPromptTemplate", MergePayload: true},
	}
	raw = append(raw, reportCRUD("stock_premarket", "盘前报告",
		"/createStockPremarketReport", "/updateStockPremarketReport", "/deleteStockPremarketReport", "/getStockPremarketReports", false)...)
	raw = append(raw, reportCRUD("stock_intraday", "盘中决策报告",
		"/createStockIntradayReport", "/updateStockIntradayReport", "/deleteStockIntradayReport", "/getStockIntradayReports", true)...)
	raw = append(raw, reportCRUD("stock_postmarket", "盘后报告",
		"/createStockPostmarketReport", "/updateStockPostmarketReport", "/deleteStockPostmarketReport", "/getStockPostmarketReports", true)...)
	raw = append(raw, botCRUD("dca_bot", botKindDCA, "DCA 交易机器人", "/createDCABot", "/updateDCABot", "/deleteDCABot", "/getAllDCABots", "/getDCABotLog")...)
	raw = append(raw, botCRUD("grid_bot", botKindGrid, "GRID 网格交易机器人", "/createGRIDBot", "/updateGRIDBot", "/deleteGRIDBot", "/getAllGRIDBots", "/getGRIDBotLog")...)
	raw = append(raw, botCRUD("smart_trade", botKindSmartTrade, "SmartTrade 机器人", "/createSmartTrade", "/updateSmartTrade", "/deleteSmartTrade", "/getAllSmartTrades", "/getSmartTradeLog")...)
	raw = append(raw, botCRUD("hdg_bot", botKindHDG, "HDG 对冲机器人", "/createHDGBot", "/updateHDGBot", "/deleteHDGBot", "/getAllHDGBots", "/getHDGBotLog")...)
	raw = append(raw, botCRUD("dca_reminder", botKindDCAReminder, "DCA 提醒机器人", "/createDCAReminder", "/updateDCAReminder", "/deleteDCAReminder", "/getAllDCAReminders", "/getDCAReminderLog")...)
	raw = append(raw, botCRUD("grid_reminder", botKindGridReminder, "GRID 提醒机器人", "/createGRIDReminder", "/updateGRIDReminder", "/deleteGRIDReminder", "/getAllGRIDReminders", "/getGRIDReminderLog")...)
	raw = append(raw, botCRUD("smart_reminder", botKindSmartReminder, "Smart 提醒机器人", "/createSmartReminder", "/updateSmartReminder", "/deleteSmartReminder", "/getAllSmartReminders", "/getSmartReminderLog")...)
	raw = append(raw, IndexSignalHTTPSpecs()...)
	raw = append(raw, CombinationSignalHTTPSpecs()...)
	raw = append(raw, MondayHTTPSpecs()...)

	out := make([]HTTPSpec, 0, len(raw))
	for _, spec := range raw {
		if BespokeNames[spec.Name] {
			continue
		}
		out = append(out, spec)
	}
	return out
}
