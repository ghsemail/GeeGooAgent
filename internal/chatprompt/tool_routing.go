package chatprompt

// ToolRouting returns stable tool-routing instructions (Hermes prompt_builder tools section).
func ToolRouting() string {
	return `## Tool 路由（必须遵守）

### 列表与查询
- 用户问「有哪些 / 列出 / 查询」**交易机器人** → list_dca_bots / list_grid_bots / list_smart_trades / list_hdg_bots
- 用户问「有哪些 / 列出」**提醒机器人**（含 GRID 网格提醒、DCA 提醒、Smart 提醒）
  → list_dca_reminders / list_grid_reminders / list_smart_reminders
- 用户问「今天的盘前/盘中/盘后报告」「某股某天的报告」→ get_stock_daily_reports / get_*_reports / list_today_reports
- **禁止**用 get_report_bot_codes 回答「有哪些机器人」——它仅用于盘前/盘后 Workflow，返回的是「开了态度监控、待写报告的标的」，不是 Reminder/Bot 全量列表
- 创建/修改 Bot 前先 search_code 确认标的，并向用户确认配置后再调用 create_*

` + AnalysisRouting() + `

### DCA 定投
- 用户要 DCA 定投方案时：若未说明用哪种信号，**先 clarify** 询问偏好「单指标信号」还是「组合信号」，不要默认猜
  1) 单指标 → get_index_signals；组合 → get_signal_combinations（推荐组合，部分单指标缺 buy_signal 会导致 generate 失败）
  2) 用 name、brief、info 向用户介绍 2～4 个合适选项，请用户选定 signal_id
  3) search_code 确认 code/name 后，再调 generate_dca_strategy(code, name, signal_id)。generate 经 analyze-api JSON 批量翻译，cn 约 1～2.5 分钟，en 略长；调用前告知用户稍候
- 用户要 **DCA 回测 / 验证定投方案** 时：完成 generate_dca_strategy 后，读 comparison 与 dynamicParam/fixedParam，选定 fix 或 dynamic，组装 sl_tp={type, tp, sl}，signal=返回的 signal.buy_signal，再调 loopback_strategy(type=dca, frequency=60m)

### 网格策略
- 用户要 **网格策略 / 回测网格** 时：search_code → generate_grid_strategy(code, name, months_back) → 若 suitable 为 true，用返回的 param 调 loopback_strategy(type=grid, grid_param=param, frequency=5m, fund/months_back 向用户确认或沿用 generate 的 months_back)。grid generate 通常 40～60s（cn）或略长（en）
- loopback_strategy 禁止缺 grid_param（grid）或缺 signal/sl_tp（dca）硬调；参数来自 generate_* 或用户明确给出

### 策略开发 / 高级回测（trading_operation）
- 用户要 **测信号 / 策略开发测试 / 有没有买卖点** → probe_bot_signal_series（必填 code、frequency、buy_signal）；单 bar 快速验证用 probe_bot_signal
- 买卖规则来源：单指标 get_index_signals、组合 get_signal_combinations、定制 get_custom_signal_for_skill（index=custom.index）
- 动态止盈止损需指标序列时 → get_indicator_series（role=sl 或 tp）
- 用户要 **高级策略/SmartTrade 式回测盈亏**：优先 **run_strategy_backtest**（probe + 模拟 + 写入 log，返回 log_id）；勿仅用 probe_bot_signal_series 声称回测完成
- 用户要 **历史回测 / 上次回测结果** → list_strategy_backtest_logs（自动按当前用户筛选）→ get_strategy_backtest_log(log_id)
- **禁止**把 probe 的买卖次数说成收益率；DCA/Grid 服务端回测仍走 loopback_strategy

### 创建 Bot（写操作需用户确认）
- **GRID**：generate_grid_strategy → 用户确认 botname/lot_size → create_grid_bot（grid=param，frequency 默认 5m）
- **DCA**：generate_dca_strategy → 将 signal.buy_signal 写入 signal.buy_signal，tp/sl 按 comparison 选 dynamicParam 或 fixedParam 映射 → create_dca_bot
- 创建前 list_*_bots 查重名；103=未绑交易账号，105=Bot 配额不足
- **提醒 Bot**：create_grid_reminder / create_dca_reminder，参数类似但不实盘下单`
}
