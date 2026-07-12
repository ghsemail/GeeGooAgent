# GeeGooBot mcp-api 接口分布总表

> **单一事实来源（SSOT）**：GeeGooBot `mcp-api` @ **3120**。
> geegoo Skill 与 GeeGoo Agent 的文档/Tool 命名应与此表对齐。

## 三层关系

```
GeeGooBot mcp-api.py  (HTTP POST /path)
        ↓
geegoo Skill (SKILL.md / ROUTING.md — 用户指令 → 调哪个 API)
        ↓
GeeGoo Agent (Tool snake_case — 自动化 workflow / chat 编排)
```

详见 [architecture.md](./architecture.md)。

## 统计

| 项目 | 数量 |
|------|------|
| mcpAPIServer HTTP 路由 | **73** |
| 文档领域 | **12** |
| GeeGoo Agent 注册 Tool | **~82**（含 HTTP + bespoke + local） |
| geegoo Skill 功能模块 | **11** Bot/Reminder + 分析/策略/Workflow |

## 领域一览

| 领域 ID | 中文 | HTTP 数 | 专题文档 | 主要消费者 |
| --- | --- | --- | --- | --- |
| common | 公共与账户 | 2 | [common.md](./common.md) | bot_manager（账户） |
| trading | 行情与资金 | 10 | [market/trading-data.md](./market/trading-data.md) | 全场景 · workflow |
| reports | 报告与 Workflow | 15 | [market/reports.md](./market/reports.md) | pre_market · intraday · post_market |
| analyst | 分析与 Prompt 模板 | 8 | [analyst/agent-analyst.md](./analyst/agent-analyst.md) | pre_market · on_demand |
| strategy | 策略生成与回测 | 3 | [strategy/README.md](./strategy/README.md) | strategy Skill |
| dca_bot | DCA 交易 Bot | 5 | [bot/dca-bot.md](./bot/dca-bot.md) | bot_manager |
| grid_bot | GRID 交易 Bot | 5 | [bot/grid-bot.md](./bot/grid-bot.md) | bot_manager |
| smart_trade | SmartTrade 交易 Bot | 5 | [bot/smart-trade.md](./bot/smart-trade.md) | bot_manager |
| hdg_bot | HDG 对冲 Bot | 5 | [bot/hdg-bot.md](./bot/hdg-bot.md) | bot_manager |
| dca_reminder | DCA 提醒 | 5 | [reminder/dca-reminder.md](./reminder/dca-reminder.md) | bot_manager |
| grid_reminder | GRID 提醒 | 5 | [reminder/grid-reminder.md](./reminder/grid-reminder.md) | bot_manager |
| smart_reminder | Smart 提醒 | 5 | [reminder/smart-reminder.md](./reminder/smart-reminder.md) | bot_manager |

## 公共与账户 (`common`)

专题文档：[common.md](./common.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/getBotLogByType` | 是 | `get_bot_log_by_type` | http | 账户 · Bot 运行日志 | post_market · bot_manager |  |
| `/getPosition` | 是 | `get_position` | http | 账户 · 持仓查询 | bot_manager |  |

## 行情与资金 (`trading`)

专题文档：[market/trading-data.md](./market/trading-data.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/checkTradingDay` | 是 | `check_trading_day` | bespoke | Workflow · 交易日 | pre_market · post_market |  |
| `/getBotYesterdayAttitude` | 是 | `get_bot_yesterday_attitude` | bespoke | Workflow · 盘前 | pre_market |  |
| `/getBroker` | 是 | `get_broker` | http | Workflow · 盘中 | intraday |  |
| `/getCapitalDistribution` | 是 | `get_capital_distribution` | bespoke | Workflow · 盘前 | pre_market |  |
| `/getCapitalFlow` | 是 | `get_capital_flow` | bespoke | Workflow · 盘前 | pre_market |  |
| `/getCurrentPrice` | 是 | `get_current_price` | bespoke | 行情 · 最新价 | pre_market · on_demand | 失败时 Agent 回退 /getTicker |
| `/getIndexSignalForSkill` | 否 | `get_index_signals` | http | 行情 · 指标信号列表 | strategy · bot_manager |  |
| `/getSignalCombinationForSkill` | 否 | `get_signal_combinations` | http | 行情 · 组合信号列表 | strategy · bot_manager |  |
| `/getTicker` | 是 | `get_ticker` | http | Workflow · 盘中 | intraday · on_demand |  |
| `/searchCode` | 否 | `search_code` | http | 行情 · 标的搜索 | on_demand · bot_manager · strategy |  |

## 报告与 Workflow (`reports`)

专题文档：[market/reports.md](./market/reports.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createIntradayTradeDecisionReport` | 是 | `create_intraday_report` | http | Workflow · 盘中决策报告 | intraday |  |
| `/createPostMarketReport` | 是 | `create_post_market_report` | http | Workflow · 盘后报告 | post_market |  |
| `/createPreMarketReport` | 是 | `create_pre_market_report` | bespoke | Workflow · 盘前报告 | pre_market |  |
| `/deleteIntradayTradeDecisionReport` | 是 | `delete_intraday_report` | http | Workflow · 盘中决策报告 | intraday |  |
| `/deletePostMarketReport` | 是 | `delete_post_market_report` | http | Workflow · 盘后报告 | post_market |  |
| `/deletePreMarketReport` | 是 | `delete_pre_market_report` | http | Workflow · 盘前报告 | pre_market |  |
| `/getIntradayTradeDecisionReports` | 是 | `get_intraday_reports` | http | Workflow · 盘中决策报告 | intraday |  |
| `/getPostMarketReports` | 是 | `get_post_market_reports` | http | Workflow · 盘后报告 | post_market |  |
| `/getPreMarketReports` | 是 | `get_pre_market_reports` | http | Workflow · 盘前报告 | pre_market |  |
| `/getReportBotCodes` | 是 | `get_report_bot_codes` | bespoke | Workflow · 报告待分析标的 | pre_market · post_market | 推荐路径；与 /getUserBotCodes 同实现 |
| `/getStockDailyReports` | 是 | `get_stock_daily_reports · list_today_reports` | bespoke | Workflow · 按日聚合查询 | pre_market · post_market | list_today_reports 幂等检查也走此接口 |
| `/getUserBotCodes` | 是 | `get_report_bot_codes` | bespoke | Workflow · 报告待分析标的 | pre_market · post_market | Deprecated，请改用 /getReportBotCodes（避免与「查 Bot 列表」混淆） |
| `/updateIntradayTradeDecisionReport` | 是 | `update_intraday_report` | http | Workflow · 盘中决策报告 | intraday |  |
| `/updatePostMarketReport` | 是 | `update_post_market_report` | http | Workflow · 盘后报告 | post_market |  |
| `/updatePreMarketReport` | 是 | `update_pre_market_report` | http | Workflow · 盘前报告 | pre_market |  |

## 分析与 Prompt 模板 (`analyst`)

专题文档：[analyst/agent-analyst.md](./analyst/agent-analyst.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createCompetitorPromptTemplate` | 是 | `create_competitor_prompt_template` | http | 股票技术面分析 | on_demand |  |
| `/createEtfPromptTemplate` | 是 | `create_etf_prompt_template` | http | 股票技术面分析 | on_demand |  |
| `/deleteCompetitorPromptTemplate` | 是 | `delete_competitor_prompt_template` | http | 股票技术面分析 | on_demand |  |
| `/deleteEtfPromptTemplate` | 是 | `delete_etf_prompt_template` | http | 股票技术面分析 | on_demand |  |
| `/editCompetitorPromptTemplate` | 是 | `edit_competitor_prompt_template` | http | 股票技术面分析 | on_demand |  |
| `/editEtfPromptTemplate` | 是 | `edit_etf_prompt_template` | http | 股票技术面分析 | on_demand |  |
| `/getMCPAnalysis` | 是 | `get_mcp_analysis` | bespoke | 股票技术面分析 | pre_market · on_demand |  |
| `/getSinglePromptTemplate` | 是 | `get_single_prompt_template` | http | 股票技术面分析 | pre_market · on_demand |  |

## 策略生成与回测 (`strategy`)

专题文档：[strategy/README.md](./strategy/README.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/generateDCAStrategy` | 否 | `generate_dca_strategy` | http | 策略生成 · DCA | strategy |  |
| `/generateGridStrategy` | 否 | `generate_grid_strategy` | http | 策略生成 · 网格 | strategy |  |
| `/loopBackStrategy` | 否 | `loopback_strategy` | http | 策略回测 | strategy |  |

## DCA 交易 Bot (`dca_bot`)

专题文档：[bot/dca-bot.md](./bot/dca-bot.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createDCABot` | 是 | `create_dca_bot` | http | DCA信号交易机器人 | bot_manager · chat |  |
| `/deleteDCABot` | 是 | `delete_dca_bot` | http | DCA信号交易机器人 | bot_manager · chat |  |
| `/getAllDCABots` | 是 | `list_dca_bots` | http | DCA信号交易机器人 | bot_manager · chat |  |
| `/getDCABotLog` | 是 | `get_dca_bot_log` | http | DCA信号交易机器人 | bot_manager · chat |  |
| `/updateDCABot` | 是 | `update_dca_bot` | http | DCA信号交易机器人 | bot_manager · chat |  |

## GRID 交易 Bot (`grid_bot`)

专题文档：[bot/grid-bot.md](./bot/grid-bot.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createGRIDBot` | 是 | `create_grid_bot` | http | GRID网格交易机器人 | bot_manager · chat |  |
| `/deleteGRIDBot` | 是 | `delete_grid_bot` | http | GRID网格交易机器人 | bot_manager · chat |  |
| `/getAllGRIDBots` | 是 | `list_grid_bots` | http | GRID网格交易机器人 | bot_manager · chat |  |
| `/getGRIDBotLog` | 是 | `get_grid_bot_log` | http | GRID网格交易机器人 | bot_manager · chat |  |
| `/updateGRIDBot` | 是 | `update_grid_bot` | http | GRID网格交易机器人 | bot_manager · chat |  |

## SmartTrade 交易 Bot (`smart_trade`)

专题文档：[bot/smart-trade.md](./bot/smart-trade.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createSmartTrade` | 是 | `create_smart_trade` | http | SmartTrade智能交易机器人 | bot_manager · chat |  |
| `/deleteSmartTrade` | 是 | `delete_smart_trade` | http | SmartTrade智能交易机器人 | bot_manager · chat |  |
| `/getAllSmartTrades` | 是 | `list_smart_trades` | http | SmartTrade智能交易机器人 | bot_manager · chat |  |
| `/getSmartTradeLog` | 是 | `get_smart_trade_log` | http | SmartTrade智能交易机器人 | bot_manager · chat |  |
| `/updateSmartTrade` | 是 | `update_smart_trade` | http | SmartTrade智能交易机器人 | bot_manager · chat |  |

## HDG 对冲 Bot (`hdg_bot`)

专题文档：[bot/hdg-bot.md](./bot/hdg-bot.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createHDGBot` | 是 | `create_hdg_bot` | http | HDG对冲交易机器人 | bot_manager · chat |  |
| `/deleteHDGBot` | 是 | `delete_hdg_bot` | http | HDG对冲交易机器人 | bot_manager · chat |  |
| `/getAllHDGBots` | 是 | `list_hdg_bots` | http | HDG对冲交易机器人 | bot_manager · chat |  |
| `/getHDGBotLog` | 是 | `get_hdg_bot_log` | http | HDG对冲交易机器人 | bot_manager · chat |  |
| `/updateHDGBot` | 是 | `update_hdg_bot` | http | HDG对冲交易机器人 | bot_manager · chat |  |

## DCA 提醒 (`dca_reminder`)

专题文档：[reminder/dca-reminder.md](./reminder/dca-reminder.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createDCAReminder` | 是 | `create_dca_reminder` | http | DCA信号提醒机器人 | bot_manager · chat |  |
| `/deleteDCAReminder` | 是 | `delete_dca_reminder` | http | DCA信号提醒机器人 | bot_manager · chat |  |
| `/getAllDCAReminders` | 是 | `list_dca_reminders` | http | DCA信号提醒机器人 | bot_manager · chat |  |
| `/getDCAReminderLog` | 是 | `get_dca_reminder_log` | http | DCA信号提醒机器人 | bot_manager · chat |  |
| `/updateDCAReminder` | 是 | `update_dca_reminder` | http | DCA信号提醒机器人 | bot_manager · chat |  |

## GRID 提醒 (`grid_reminder`)

专题文档：[reminder/grid-reminder.md](./reminder/grid-reminder.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createGRIDReminder` | 是 | `create_grid_reminder` | http | GRID网格提醒机器人 | bot_manager · chat |  |
| `/deleteGRIDReminder` | 是 | `delete_grid_reminder` | http | GRID网格提醒机器人 | bot_manager · chat |  |
| `/getAllGRIDReminders` | 是 | `list_grid_reminders` | http | GRID网格提醒机器人 | bot_manager · chat |  |
| `/getGRIDReminderLog` | 是 | `get_grid_reminder_log` | http | GRID网格提醒机器人 | bot_manager · chat |  |
| `/updateGRIDReminder` | 是 | `update_grid_reminder` | http | GRID网格提醒机器人 | bot_manager · chat |  |

## Smart 提醒 (`smart_reminder`)

专题文档：[reminder/smart-reminder.md](./reminder/smart-reminder.md)

| HTTP | mcp_token | GeeGoo Agent Tool | Tool 类型 | geegoo Skill | Agent 场景 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `/createSmartReminder` | 是 | `create_smart_reminder` | http | SmartTrade交易提醒机器人 | bot_manager · chat |  |
| `/deleteSmartReminder` | 是 | `delete_smart_reminder` | http | SmartTrade交易提醒机器人 | bot_manager · chat |  |
| `/getAllSmartReminders` | 是 | `list_smart_reminders` | http | SmartTrade交易提醒机器人 | bot_manager · chat |  |
| `/getSmartReminderLog` | 是 | `get_smart_reminder_log` | http | SmartTrade交易提醒机器人 | bot_manager · chat |  |
| `/updateSmartReminder` | 是 | `update_smart_reminder` | http | SmartTrade交易提醒机器人 | bot_manager · chat |  |

## GeeGoo Agent 本地 Tool（无 MCP HTTP）

| Tool | 类型 | geegoo Skill | Agent 场景 |
| --- | --- | --- | --- |
| `fetch_market_news` | local | finance-news Skill | pre_market |
| `fetch_stock_news` | local | finance-news Skill | pre_market |
| `save_local_report` | local | Workflow · 本地 Markdown | pre_market · post_market |
| `send_feishu_summary` | local | Workflow · 通知 | pre_market |
| `write_execution_log` | local | Workflow · 审计 | pre_market |
| `recall` | local | chat 会话记忆 | chat |
| `recall_yesterday_summary` | local | Workflow · 记忆 | pre_market |
| `read_working_state` | local | Workflow · 状态 | pre_market |

## 已知缺口

| HTTP（预期） | Agent Tool | 说明 |
| --- | --- | --- |
| `/switchBot` | `switch_bot` | catalog 已注册，mcpAPIServer 尚无此路由 |

---

由 `scripts/generate_interface_map.py` 生成；修改映射后重新运行该脚本。
