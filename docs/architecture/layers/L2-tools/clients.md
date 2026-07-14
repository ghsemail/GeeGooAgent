# L2 — HTTP Clients

## 三 Client 架构

```text
Tools
  ├── MarketClient     → :3120  (sk-...)   GeeGooBot mcp-api workflow
  ├── GeeGooBotClient    → :3120  (sk-...)   geegoo 主 API
  └── SignalClient     → :3210             信号指标（可选）
```

配置：

```json
{
  "base_url": "http://118.195.135.97:3120",
  "api_key": "sk-...",
  "geegoo_url": "http://118.195.135.97:3120",
  "geegoo_api_key": "sk-...",
  "signal_base_url": "http://146.56.225.252:3210"
}
```

## BaseClient

- Header：`Authorization: Bearer`, `Content-Type: application/json`
- 重试：3×，间隔 5s
- **Network allowlist**（Sandbox L2）：仅允许配置内 host
- 凭证：`SecretsProvider` 注入

## GeeGooBotClient（3120）— geegoo

| 方法                         | 路径                                 | MVP            |
| -------------------------- | ---------------------------------- | -------------- |
| check_trading_day          | /checkTradingDay                   | ✓              |
| get_report_bot_codes         | /getReportBotCodes                   | ✓              |
| search_code                | /searchCode                        |                |
| get_capital_flow           | /getCapitalFlow                    | ✓              |
| get_capital_distribution   | /getCapitalDistribution            | ✓              |
| get_bot_yesterday_attitude | /getBotYesterdayAttitude           | ✓              |
| get_bot_log_by_type        | /getBotLogByType                   | Phase 2        |
| get_single_prompt_template | /getSinglePromptTemplate           |                |
| create_pre_market_report   | /createPreMarketReport             | ✓              |
| update_pre_market_report   | /updatePreMarketReport             |                |
| create_post_market_report  | /createPostMarketReport            |                |
| create_intraday_report     | /createIntradayTradeDecisionReport |                |
| get_ticker                 | /getTicker                         | Phase 3        |
| get_broker                 | /getBroker                         | Phase 3        |
| get_mcp_analysis           | /getMCPAnalysis                    | ✓（盘前 workflow） |
| get_pre_market_reports     | /getPreMarketReports               | Phase 2        |
| delete_pre_market_report   | /deletePreMarketReport             |                |
| get_post_market_reports    | /getPostMarketReports              |                |
| get_intraday_reports       | /getIntradayTradeDecisionReports   |                |

## ReportClient（6100）— reportServer（可选）

| 方法                        | 路径             | 说明                                       |
| ------------------------- | -------------- | ---------------------------------------- |
| get_daily_reports_unified | /reports/daily | Bearer `REPORT_SERVER_API_KEY` + user_id |

## GeeGooBotClient（3120）— geegoo

### 分析 & 报告

| 方法                         | 路径                       |
| -------------------------- | ------------------------ |
| get_mcp_analysis           | /getMCPAnalysis          |
| get_single_prompt_template | /getSinglePromptTemplate |
| get_tech_prompt_list       | /getTechPromptList       |
| get_stock_daily_reports    | /getStockDailyReports    |

### 市场数据

| 方法                | 路径               | mcp_token |
| ----------------- | ---------------- | --------- |
| search_code       | /searchCode      | 否         |
| get_position      | /getPosition     | 是         |
| get_current_price | /getCurrentPrice | 否         |

### 策略

| 方法                      | 路径                            |
| ----------------------- | ----------------------------- |
| generate_grid_strategy  | /generateGridStrategy         |
| generate_dca_strategy   | /generateDCAStrategy          |
| loopback_strategy       | /loopBackStrategy             |
| get_index_signals       | /getIndexSignalForSkill       |
| get_signal_combinations | /getSignalCombinationForSkill |

### Bot / Reminder CRUD（Phase 6 扩展）

每组四件套 `create`_* / `update_`* / `delete_`* / `list_`*：

- DCA Bot → `/createDCABot` … `/getAllDCABots`
- GRID Bot → `/createGRIDBot` …
- SmartTrade → `/createSmartTrade` …
- HDG → `/createHDGBot` …
- DCA Reminder → `/createDCAReminder` …
- GRID Reminder → `/createGRIDReminder` …
- Smart Reminder → `/createSmartReminder` …

### Bot 日志（Phase 6）

- get_dca_bot_log → `/getDCABotLog`
- get_grid_bot_log → `/getGRIDBotLog`
- get_smart_trade_log → `/getSmartTradeLog`
- get_dca_reminder_log → `/getDCAReminderLog`
- get_grid_reminder_log → `/getGRIDReminderLog`
- get_smart_reminder_log → `/getSmartReminderLog`

### Prompt 模板 CRUD（Phase 7）

- Competitor / Etf 各 create/edit/delete

## SignalClient（3210 / 3200）

若 3120 未代理信号接口，则：

- get_index_signals → `{signal_base_url}/getIndexSignalForSkill`
- get_signal_combinations → `{signal_base_url}/getSignalCombinationForSkill`

## 接口路由说明（2026-05-20 后）

以下两个接口**历史上曾有服务端 bug，均已修复**，应正常封装为 Client 方法：

| 接口                      | 历史问题                               | 当前状态    | 推荐用法                                                |
| ----------------------- | ---------------------------------- | ------- | --------------------------------------------------- |
| `/getCapitalFlow`       | `PeriodType.YEAR` 枚举缺失导致全部 500     | ✅ 已修复   | 盘前 `period=DAY`；与 `getCapitalDistribution` **同时调用** |
| `/getPreMarketReports`  | `bot_id` ObjectId 未 JSON 序列化导致 500 | ✅ 已修复   | 5900 按 code/report_id 查盘前报告；盘后兜底                    |
| `/getStockDailyReports` | —                                  | 3120 正常 | **按日期聚合** pre/intraday/post 时优先用此接口                 |

## 代码路径

`src/geegoo/clients/market.py`, `geegoo_bot.py`, `signal.py`, `base.py`