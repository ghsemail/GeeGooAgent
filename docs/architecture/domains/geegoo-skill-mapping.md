# geegoo skill → Tool 完整映射

对照 [`geegoo/SKILL.md`](C:\Users\ghsemail\.cursor\skills\geegoo\SKILL.md) 与 [`geegoo/docs/`](C:\Users\ghsemail\.cursor\skills\geegoo\docs)。

## SKILL 章节 → Tool 组

| SKILL 章节 | MCP 接口 | GeeGoo Agent Tool | Phase |
|------------|----------|-----------------|-------|
| 股票技术面分析 | getTechPromptList, getSinglePromptTemplate, getMCPAnalysis | get_tech_prompt_list, get_single_prompt_template, get_mcp_analysis | 1/4 |
| 股票日常报告查询 | getStockDailyReports | get_stock_daily_reports, list_today_reports | 1 |
| 信号指标 | getIndexSignalForSkill, getSignalCombinationForSkill | get_index_signals, get_signal_combinations | 5 |
| 策略生成 | generateGridStrategy, generateDCAStrategy | generate_grid_strategy, generate_dca_strategy | 5 |
| 策略回测 | loopBackStrategy | loopback_strategy | 5 |
| DCA 信号交易机器人 | create/update/delete/getAll DCABot + getDCABotLog | create/update/delete/list_dca_bot, get_dca_bot_log | 6 |
| SmartTrade 交易机器人 | create/update/delete/getAll SmartTrade + getSmartTradeLog | create/update/delete/list_smart_trade, get_smart_trade_log | 6 |
| GRID 网格交易机器人 | create/update/delete/getAll GRIDBot + getGRIDBotLog | create/update/delete/list_grid_bot, get_grid_bot_log | 6 |
| HDG 对冲交易机器人 | create/update/delete/getAll HDGBot | create/update/delete/list_hdg_bot | 6 |
| DCA 信号提醒机器人 | create/update/delete/getAll DCAReminder + getDCAReminderLog | create/update/delete/list_dca_reminder, get_dca_reminder_log | 6 |
| GRID 网格提醒机器人 | create/update/delete/getAll GRIDReminder + getGRIDReminderLog | idem grid_reminder | 6 |
| SmartTrade 交易提醒 | create/update/delete/getAll SmartReminder + getSmartReminderLog | idem smart_reminder | 6 |
| common / trading | searchCode, getPosition, getCurrentPrice, getReportBotCodes | search_code, get_position, get_current_price, get_report_bot_codes | 1–6 |
| AgentAnalyst 竞品/ETF | create/edit/delete Competitor/Etf PromptTemplate | §2.4 六个 tool | 7 |

## 端口

| 端口 | 用途 | Client |
|------|------|--------|
| **5700** | geegoo mcp（全部 HTTP） | `GeeGooBotClient` / `MarketClient` 别名 |
| **5800** | 信号服务（可选 `signal_base_url`） | `SignalClient` |

HTTP ↔ Tool 总表：[interface-map.md](../../reference/geegoo-mcp/interface-map.md)

## 交互规范（→ rules/bot-creation.md）

1. 任何 `create_*_bot` / `create_*_reminder` 前必须 `search_code`
2. 展示完整方案 → `wait_for_human` → 再 create
3. GRID Reminder 创建时显式 `frequency: 60m`
4. SmartTrade sell_only：创建前 `get_position`，不传 price

## clients 移植

[`geegoobot_client.py`](C:\Users\ghsemail\.cursor\skills\geegoo\scripts\geegoobot_client.py) 仅实现 DCA Reminder CRUD + 信号查询 → Phase 6 需扩展全量 Bot/Reminder 方法。

完整 Tool 列表见 [../layers/L2-tools/tool-catalog.md](../layers/L2-tools/tool-catalog.md)。
