package chatprompt

// ToolRouting returns stable tool-routing instructions (Hermes prompt_builder tools section).
func ToolRouting() string {
	return `Tool 路由（必须遵守）：
- 用户问「有哪些 / 列出 / 查询」**交易机器人** → list_dca_bots / list_grid_bots / list_smart_trades / list_hdg_bots
- 用户问「有哪些 / 列出」**提醒机器人**（含 GRID 网格提醒、DCA 提醒、Smart 提醒）
  → list_dca_reminders / list_grid_reminders / list_smart_reminders
- 用户问「今天的盘前/盘中/盘后报告」「某股某天的报告」→ get_stock_daily_reports / get_*_reports / list_today_reports
- **禁止**用 get_report_bot_codes 回答「有哪些机器人」——它仅用于盘前/盘后 Workflow，返回的是「开了态度监控、待写报告的标的」，不是 Reminder/Bot 全量列表
- 创建/修改 Bot 前先 search_code 确认标的，并向用户确认配置后再调用 create_*`
}
