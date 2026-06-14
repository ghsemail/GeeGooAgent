# Tool 服务器与接口对照表

> 维护说明：本文档供人工编辑。Tool 注册以 `src/geegoo/tools/catalog.py` 与 bespoke 实现为准；改 API 路由时同步更新此表。
>
> 生成基准：`geegoo` 全量 82 个 Tool（`bootstrap.all_tool_instances()`）。

## 服务器别名

| 别名 | IP | 端口 | 说明 |
|------|-----|------|------|
| **geegoo mcp** | `118.195.135.97` | **5700** | 统一 MCP API（Bot CRUD、策略、分析、workflow、报告） |
| **SignalServer** | `146.56.225.252` | **5800** | 信号/回测（Agent **不直连**，经 geegoo mcp 转发） |
| **TradingServer** | `43.134.94.87` | **7000** | 富途行情/持仓（Agent **不直连**，经 geegoo mcp 转发） |

> **5900 已废弃**：原 geegoo mcp 已合并入 geegoo mcp（5700）。

配置项对应关系（`config.json`）：

| 配置项               | 默认 URL                       | 说明                                    |
| ----------------- | ---------------------------- | --------------------------------------- |
| `geegoo_url`        | `http://118.195.135.97:5700` | geegoo mcp 唯一入口                            |
| `geegoo_api_key`    | `sk-...`                     | geegoo mcp Bearer Token（`mk-` 已废弃）       |
| `base_url` / `api_key` | 同 `geegoo_url` / `geegoo_api_key` | 兼容旧配置，新部署请只用 geegoo_* |
| `signal_base_url` | `http://146.56.225.252:5800` | SignalServer（Tool 不直连） |

## 图例

| 列 | 含义 |
|----|------|
| **直连服务器** | Agent HTTP 直接访问的服务器 |
| **间接服务器** | geegoo mcp 服务端内部再调用的下游（Agent 不直连） |
| **接口服务** | geegoo mcp 上的具体 API 进程与端口 |
| **接口方法** | HTTP 方法与路径（均为 `POST`） |

特殊值：

- **—** — 不依赖 geegoo mcp / SignalServer / TradingServer
- **外网 HTTPS** — 新闻脚本或飞书 Webhook 等公网出站

---

## 按服务器汇总

| 服务器 | 涉及 Tool 数 | 说明 |
|--------|-------------|------|
| **geegoo mcp** | 74 | 全部 HTTP Tool 的最终入口 |
| **SignalServer**（间接） | 5 | 信号列表、组合信号、策略生成、回测 |
| **TradingServer**（间接） | 1～2 | 最新价、持仓（5700 路径，服务端转发） |
| **无（本地/外网）** | 8 | workspace、chat 文件、新闻、飞书 |

---

## 间接依赖速查

| 间接服务器 | 触发 Tool | geegoo mcp 入口 |
|-----------|----------|----------------|
| **TradingServer** `:7000` | `get_current_price` | geegoo mcp `POST /getCurrentPrice` |
| **TradingServer** `:7000` | `get_position` | geegoo mcp `POST /getPosition` |
| **SignalServer** `:5800` | `get_index_signals` | geegoo mcp `POST /getIndexSignalForSkill` |
| **SignalServer** `:5800` | `get_signal_combinations` | geegoo mcp `POST /getSignalCombinationForSkill` |
| **SignalServer** `:5800` | `generate_grid_strategy` | geegoo mcp `POST /generateGridStrategy` |
| **SignalServer** `:5800` | `generate_dca_strategy` | geegoo mcp `POST /generateDCAStrategy` |
| **SignalServer** `:5800` | `loopback_strategy` | geegoo mcp `POST /loopBackStrategy` |

---

## 一、感知 Perception

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `check_trading_day` | geegoo mcp | — | geegoo mcp:5700 | `POST /checkTradingDay` |
| `search_code` | geegoo mcp | — | geegoo mcp:5700 | `POST /searchCode` |
| `get_current_price` | geegoo mcp | TradingServer | geegoo mcp:5700 | `POST /getCurrentPrice`；失败回退 `POST /getTicker` |
| `get_ticker` | geegoo mcp | — | geegoo mcp:5700 | `POST /getTicker` |
| `get_position` | geegoo mcp | TradingServer | geegoo mcp:5700 | `POST /getPosition` |
| `get_broker` | geegoo mcp | — | geegoo mcp:5700 | `POST /getBroker` |
| `get_report_bot_codes` | geegoo mcp | — | geegoo mcp:5700 | `POST /getReportBotCodes` |
| `fetch_market_news` | — | 外网 HTTPS | 本地脚本 | subprocess（东方财富 / finance-news） |
| `fetch_stock_news` | — | 外网 HTTPS | 本地脚本 | subprocess（东方财富 / akshare / finance-news） |

---

## 二、分析 Analysis

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `get_mcp_analysis` | geegoo mcp | — | geegoo mcp:5700 | `POST /getMCPAnalysis` |
| `get_capital_flow` | geegoo mcp | — | geegoo mcp:5700 | `POST /getCapitalFlow` |
| `get_capital_distribution` | geegoo mcp | — | geegoo mcp:5700 | `POST /getCapitalDistribution` |
| `get_bot_yesterday_attitude` | geegoo mcp | — | geegoo mcp:5700 | `POST /getBotYesterdayAttitude` |
| `get_bot_log_by_type` | geegoo mcp | — | geegoo mcp:5700 | `POST /getBotLogByType` |
| `get_stock_daily_reports` | geegoo mcp | — | geegoo mcp:5700 | `POST /getStockDailyReports` |
| `list_today_reports` | geegoo mcp | — | geegoo mcp:5700 | `POST /getStockDailyReports` |
| `get_pre_market_reports` | geegoo mcp | — | geegoo mcp:5700 | `POST /getPreMarketReports` |
| `get_intraday_reports` | geegoo mcp | — | geegoo mcp:5700 | `POST /getIntradayTradeDecisionReports` |
| `get_post_market_reports` | geegoo mcp | — | geegoo mcp:5700 | `POST /getPostMarketReports` |
| `get_index_signals` | geegoo mcp | SignalServer | geegoo mcp:5700 | `POST /getIndexSignalForSkill` |
| `get_signal_combinations` | geegoo mcp | SignalServer | geegoo mcp:5700 | `POST /getSignalCombinationForSkill` |
| `get_single_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /getSinglePromptTemplate` |
| `generate_grid_strategy` | geegoo mcp | SignalServer | geegoo mcp:5700 | `POST /generateGridStrategy` |
| `generate_dca_strategy` | geegoo mcp | SignalServer | geegoo mcp:5700 | `POST /generateDCAStrategy` |
| `loopback_strategy` | geegoo mcp | SignalServer | geegoo mcp:5700 | `POST /loopBackStrategy` |
| `get_dca_bot_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getDCABotLog` |
| `get_grid_bot_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getGRIDBotLog` |
| `get_smart_trade_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getSmartTradeLog` |
| `get_hdg_bot_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getHDGBotLog` |
| `get_dca_reminder_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getDCAReminderLog` |
| `get_grid_reminder_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getGRIDReminderLog` |
| `get_smart_reminder_log` | geegoo mcp | — | geegoo mcp:5700 | `POST /getSmartReminderLog` |

---

## 三、决策 Decision

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `recall_yesterday_summary` | — | — | 本地 workspace | 读 `reports/{date}/{code}-premarket.md` |

---

## 四、元数据 Meta

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `recall` | — | — | 本地 chat 存储 | 搜索 `chat/{session_id}.json` |
| `read_working_state` | — | — | 本地 workspace | 读 `working/{session_id}.json` |
| `write_execution_log` | — | — | 本地 workspace | 写 `execution-log.md` |

---

## 五、行动 Action

### 5.1 报告

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_pre_market_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /createPreMarketReport` |
| `update_pre_market_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /updatePreMarketReport` |
| `delete_pre_market_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /deletePreMarketReport` |
| `create_intraday_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /createIntradayTradeDecisionReport` |
| `update_intraday_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateIntradayTradeDecisionReport` |
| `delete_intraday_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteIntradayTradeDecisionReport` |
| `create_post_market_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /createPostMarketReport` |
| `update_post_market_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /updatePostMarketReport` |
| `delete_post_market_report` | geegoo mcp | — | geegoo mcp:5700 | `POST /deletePostMarketReport` |
| `save_local_report` | — | — | 本地 workspace | 写 `reports/{date}/{code}-*.md` |
| `send_feishu_summary` | — | 外网 HTTPS | 飞书 Webhook | `POST feishu_webhook_url` |

### 5.2 Prompt 模板

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_competitor_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /createCompetitorPromptTemplate` |
| `edit_competitor_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /editCompetitorPromptTemplate` |
| `delete_competitor_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteCompetitorPromptTemplate` |
| `create_etf_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /createEtfPromptTemplate` |
| `edit_etf_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /editEtfPromptTemplate` |
| `delete_etf_prompt_template` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteEtfPromptTemplate` |

### 5.3 交易 Bot（DCA / GRID / SmartTrade / HDG）

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_dca_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /createDCABot` |
| `update_dca_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateDCABot` |
| `delete_dca_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteDCABot` |
| `list_dca_bots` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllDCABots` |
| `create_grid_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /createGRIDBot` |
| `update_grid_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateGRIDBot` |
| `delete_grid_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteGRIDBot` |
| `list_grid_bots` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllGRIDBots` |
| `create_smart_trade` | geegoo mcp | — | geegoo mcp:5700 | `POST /createSmartTrade` |
| `update_smart_trade` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateSmartTrade` |
| `delete_smart_trade` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteSmartTrade` |
| `list_smart_trades` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllSmartTrades` |
| `create_hdg_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /createHDGBot` |
| `update_hdg_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateHDGBot` |
| `delete_hdg_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteHDGBot` |
| `list_hdg_bots` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllHDGBots` |

### 5.4 提醒 Bot（DCA / GRID / Smart Reminder）

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_dca_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /createDCAReminder` |
| `update_dca_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateDCAReminder` |
| `delete_dca_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteDCAReminder` |
| `list_dca_reminders` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllDCAReminders` |
| `create_grid_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /createGRIDReminder` |
| `update_grid_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateGRIDReminder` |
| `delete_grid_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteGRIDReminder` |
| `list_grid_reminders` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllGRIDReminders` |
| `create_smart_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /createSmartReminder` |
| `update_smart_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /updateSmartReminder` |
| `delete_smart_reminder` | geegoo mcp | — | geegoo mcp:5700 | `POST /deleteSmartReminder` |
| `list_smart_reminders` | geegoo mcp | — | geegoo mcp:5700 | `POST /getAllSmartReminders` |

### 5.5 其他

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `switch_bot` | geegoo mcp | — | geegoo mcp:5700 | `POST /switchBot` |

---

## geegoo mcp 统一入口（5700）

| 配置项 | URL | 说明 |
|--------|-----|------|
| `geegoo_url` / `geegoo_api_key` | `http://118.195.135.97:5700` | 全部 HTTP Tool |
| `base_url` / `api_key` | 同上（兼容旧字段） | 与 `geegoo_*` 保持一致即可 |

---

## 场景子集

### `geegoo chat`（15 个）

| Tool | 直连 | 间接 | 接口 |
|------|------|------|------|
| `check_trading_day` | geegoo mcp | — | geegoo mcp `POST /checkTradingDay` |
| `search_code` | geegoo mcp | — | geegoo mcp `POST /searchCode` |
| `get_current_price` | geegoo mcp | TradingServer | geegoo mcp `POST /getCurrentPrice` → 回退 geegoo mcp `POST /getTicker` |
| `get_ticker` | geegoo mcp | — | geegoo mcp `POST /getTicker` |
| `get_position` | geegoo mcp | TradingServer | geegoo mcp `POST /getPosition` |
| `get_capital_flow` | geegoo mcp | — | geegoo mcp `POST /getCapitalFlow` |
| `get_capital_distribution` | geegoo mcp | — | geegoo mcp `POST /getCapitalDistribution` |
| `get_single_prompt_template` | geegoo mcp | — | geegoo mcp `POST /getSinglePromptTemplate` |
| `get_mcp_analysis` | geegoo mcp | — | geegoo mcp `POST /getMCPAnalysis` |
| `fetch_market_news` | — | 外网 | 本地脚本 |
| `fetch_stock_news` | — | 外网 | 本地脚本 |
| `get_report_bot_codes` | geegoo mcp | — | geegoo mcp `POST /getReportBotCodes` |
| `get_stock_daily_reports` | geegoo mcp | — | geegoo mcp `POST /getStockDailyReports` |
| `list_today_reports` | geegoo mcp | — | geegoo mcp `POST /getStockDailyReports` |
| `write_execution_log` | — | — | 本地 workspace |
| `recall` | — | — | 本地 chat 存储 |

### 盘前 workflow（16 个，见 `skills/pre_market/manifest.yaml`）

`check_trading_day`, `get_report_bot_codes`, `fetch_market_news`, `fetch_stock_news`, `get_mcp_analysis`, `get_stock_daily_reports`, `list_today_reports`, `get_capital_flow`, `get_capital_distribution`, `get_bot_yesterday_attitude`, `recall_yesterday_summary`, `read_working_state`, `create_pre_market_report`, `save_local_report`, `send_feishu_summary`, `write_execution_log`

---

## 相关文档

- [tool-catalog.md](./tool-catalog.md) — Tool 功能与 MVP 标记
- [clients.md](./clients.md) — HTTP Client 设计
- [geegoo-api-routing.md](../../domains/geegoo-api-routing.md) — geegoo mcp 5700 路由规则
