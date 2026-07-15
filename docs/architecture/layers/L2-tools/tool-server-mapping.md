# Tool 服务器与接口对照表

> 维护：Tool 注册以 `internal/tools/catalog/catalog.go` 与 `bespoke.go` 为准；改 API 时同步本表与 [interface-map.md](../../../reference/geegoo-mcp/interface-map.md)。  
> 运行态可用性：[tools-tree.md](./tools-tree.md)

## 服务器别名（GeeGoo Go 3xxx）

| 别名 | IP | 端口 | 说明 |
|------|-----|------|------|
| **GeeGooBot mcp-api** | `118.195.135.97` | **3120** | MCP 契约 API（GeeGooAgent 主入口） |
| **GeeGooSignal signal-api** | `146.56.225.252` | **3200** | 标的搜索、回测等 |
| **GeeGooSignal catalog-api** | `146.56.225.252` | **3210** | 指标信号、queryModel |
| **GeeGooSignal analyze-api** | `146.56.225.252` | **3230** | MCP 分析（getMCPAnalysis） |
| **GeeGooData data-api** | `47.80.14.120` | **3300** | 行情数据 |

配置项对应关系（`config.json`）：

| 配置项 | 生产 URL | 说明 |
| ----------------- | ---------------------------- | --------------------------------------- |
| `geegoo_url` / `base_url` | `http://118.195.135.97:3120` | GeeGooBot mcp-api |
| `geegoo_api_key` / `api_key` | `GEEGOO_BOT_MCP_API_KEY` | mcp-api Bearer（**非**旧 Python sk- key） |
| `signal_base_url` | `http://146.56.225.252:3210` | catalog-api |
| `signal_api_url` | `http://146.56.225.252:3200` | signal-api（可省略，由 catalog 主机推导） |
| `signal_catalog_api_key` | 各服务 Bearer | 见 GeeGooSignal `.env` |
| `signal_analyze_api_url` | `http://146.56.225.252:3230` | analyze-api |
| `signal_analyze_api_key` | analyze-api Bearer | 见 GeeGooSignal `.env` |
| `data_base_url` | `http://47.80.14.120:3300` | GeeGooData |

参考：`config.production.example.json`

## 图例

| 列 | 含义 |
|----|------|
| **直连服务器** | Agent HTTP 直接访问的服务器 |
| **间接服务器** | GeeGooBot mcp-api 服务端内部再调用的下游（Agent 不直连） |
| **接口服务** | GeeGooBot mcp-api 上的具体 API 进程与端口 |
| **接口方法** | HTTP 方法与路径（均为 `POST`） |

特殊值：

- **—** — 不依赖 GeeGooBot mcp-api / GeeGooSignal / TradingServer
- **外网 HTTPS** — 新闻脚本或飞书 Webhook 等公网出站

---

## 按服务器汇总

| 服务器 | 涉及 Tool 数 | 说明 |
|--------|-------------|------|
| **GeeGooBot mcp-api** | 74 | 全部 HTTP Tool 的最终入口 |
| **GeeGooSignal**（间接） | 5 | 信号列表、组合信号、策略生成、回测 |
| **TradingServer**（间接） | 1～2 | 最新价、持仓（GeeGooBot mcp-api 内部转发） |
| **无（本地/外网）** | 8 | workspace、chat 文件、新闻、飞书 |

---

## 间接依赖速查

| 间接服务器 | 触发 Tool | GeeGooBot mcp-api 入口 |
|-----------|----------|----------------|
| **TradingServer** `:7000` | `get_current_price` | GeeGooBot mcp-api `POST /getCurrentPrice` |
| **TradingServer** `:7000` | `get_position` | GeeGooBot mcp-api `POST /getPosition` |
| **GeeGooSignal** `:3210` | `get_index_signals` | GeeGooBot mcp-api `POST /getIndexSignalForSkill` |
| **GeeGooSignal** `:3210` | `get_signal_combinations` | GeeGooBot mcp-api `POST /getSignalCombinationForSkill` |
| **GeeGooSignal** `:3210` | `generate_grid_strategy` | GeeGooBot mcp-api `POST /generateGridStrategy` |
| **GeeGooSignal** `:3210` | `generate_dca_strategy` | GeeGooBot mcp-api `POST /generateDCAStrategy` |
| **GeeGooSignal signal-api** `:3200` | `loopback_strategy` | GeeGooSignal signal-api `POST /loopBackStrategy` |

---

## 一、感知 Perception

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `check_trading_day` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /checkTradingDay` |
| `search_code` | GeeGooSignal signal-api | — | GeeGooSignal signal-api:3200 | `POST /searchCode` |
| `get_current_price` | GeeGooBot mcp-api | TradingServer | GeeGooBot mcp-api:3120 | `POST /getCurrentPrice`；失败回退 `POST /getTicker` |
| `get_ticker` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getTicker` |
| `get_position` | GeeGooBot mcp-api | TradingServer | GeeGooBot mcp-api:3120 | `POST /getPosition` |
| `get_broker` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getBroker` |
| `get_report_bot_codes` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getReportBotCodes` |
| `fetch_market_news` | — | 外网 HTTPS | 本地脚本 | subprocess（东方财富 / finance-news） |
| `fetch_stock_news` | — | 外网 HTTPS | 本地脚本 | subprocess（东方财富 / akshare / finance-news） |

---

## 二、分析 Analysis

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `get_mcp_analysis` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getMCPAnalysis` |
| `get_capital_flow` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getCapitalFlow` |
| `get_capital_distribution` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getCapitalDistribution` |
| `get_bot_yesterday_attitude` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getBotYesterdayAttitude` |
| `get_bot_log_by_type` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getBotLogByType` |
| `get_stock_daily_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getStockDailyReports` |
| `list_today_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getStockDailyReports` |
| `get_pre_market_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getPreMarketReports` |
| `get_intraday_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getIntradayTradeDecisionReports` |
| `get_post_market_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getPostMarketReports` |
| `get_index_signals` | GeeGooBot mcp-api | GeeGooSignal | GeeGooBot mcp-api:3120 | `POST /getIndexSignalForSkill` |
| `get_signal_combinations` | GeeGooBot mcp-api | GeeGooSignal | GeeGooBot mcp-api:3120 | `POST /getSignalCombinationForSkill` |
| `get_single_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getSinglePromptTemplate` |
| `generate_grid_strategy` | GeeGooBot mcp-api | GeeGooSignal | GeeGooBot mcp-api:3120 | `POST /generateGridStrategy` |
| `generate_dca_strategy` | GeeGooBot mcp-api | GeeGooSignal | GeeGooBot mcp-api:3120 | `POST /generateDCAStrategy` |
| `loopback_strategy` | GeeGooBot mcp-api | GeeGooSignal | GeeGooBot mcp-api:3120 | `POST /loopBackStrategy` |
| `get_dca_bot_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getDCABotLog` |
| `get_grid_bot_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getGRIDBotLog` |
| `get_smart_trade_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getSmartTradeLog` |
| `get_hdg_bot_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getHDGBotLog` |
| `get_dca_reminder_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getDCAReminderLog` |
| `get_grid_reminder_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getGRIDReminderLog` |
| `get_smart_reminder_log` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getSmartReminderLog` |

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
| `create_pre_market_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createPreMarketReport` |
| `update_pre_market_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updatePreMarketReport` |
| `delete_pre_market_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deletePreMarketReport` |
| `create_intraday_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createIntradayTradeDecisionReport` |
| `update_intraday_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateIntradayTradeDecisionReport` |
| `delete_intraday_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteIntradayTradeDecisionReport` |
| `create_post_market_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createPostMarketReport` |
| `update_post_market_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updatePostMarketReport` |
| `delete_post_market_report` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deletePostMarketReport` |
| `save_local_report` | — | — | 本地 workspace | 写 `reports/{date}/{code}-*.md` |
| `send_feishu_summary` | — | 外网 HTTPS | 飞书 Webhook | `POST feishu_webhook_url` |

### 5.2 Prompt 模板

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_competitor_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createCompetitorPromptTemplate` |
| `edit_competitor_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /editCompetitorPromptTemplate` |
| `delete_competitor_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteCompetitorPromptTemplate` |
| `create_etf_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createEtfPromptTemplate` |
| `edit_etf_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /editEtfPromptTemplate` |
| `delete_etf_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteEtfPromptTemplate` |

### 5.3 交易 Bot（DCA / GRID / SmartTrade / HDG）

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_dca_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createDCABot` |
| `update_dca_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateDCABot` |
| `delete_dca_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteDCABot` |
| `list_dca_bots` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllDCABots` |
| `create_grid_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createGRIDBot` |
| `update_grid_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateGRIDBot` |
| `delete_grid_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteGRIDBot` |
| `list_grid_bots` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllGRIDBots` |
| `create_smart_trade` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createSmartTrade` |
| `update_smart_trade` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateSmartTrade` |
| `delete_smart_trade` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteSmartTrade` |
| `list_smart_trades` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllSmartTrades` |
| `create_hdg_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createHDGBot` |
| `update_hdg_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateHDGBot` |
| `delete_hdg_bot` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteHDGBot` |
| `list_hdg_bots` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllHDGBots` |

### 5.4 提醒 Bot（DCA / GRID / Smart Reminder）

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `create_dca_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createDCAReminder` |
| `update_dca_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateDCAReminder` |
| `delete_dca_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteDCAReminder` |
| `list_dca_reminders` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllDCAReminders` |
| `create_grid_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createGRIDReminder` |
| `update_grid_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateGRIDReminder` |
| `delete_grid_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteGRIDReminder` |
| `list_grid_reminders` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllGRIDReminders` |
| `create_smart_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /createSmartReminder` |
| `update_smart_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /updateSmartReminder` |
| `delete_smart_reminder` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /deleteSmartReminder` |
| `list_smart_reminders` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getAllSmartReminders` |

### 5.5 其他

（无额外 HTTP Tool；`write_execution_log` 等为本地 bespoke。）

---

## GeeGooBot mcp-api 统一入口（3120）

| 配置项 | URL | 说明 |
|--------|-----|------|
| `geegoo_url` / `geegoo_api_key` | `http://118.195.135.97:3120` | 全部 HTTP Tool |
| `base_url` / `api_key` | 同上（兼容旧字段） | 与 `geegoo_*` 保持一致即可 |

---

## 场景子集

### `geegoo chat`（15 个）

| Tool | 直连 | 间接 | 接口 |
|------|------|------|------|
| `check_trading_day` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /checkTradingDay` |
| `search_code` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /searchCode` |
| `get_current_price` | GeeGooBot mcp-api | TradingServer | GeeGooBot mcp-api `POST /getCurrentPrice` → 回退 GeeGooBot mcp-api `POST /getTicker` |
| `get_ticker` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getTicker` |
| `get_position` | GeeGooBot mcp-api | TradingServer | GeeGooBot mcp-api `POST /getPosition` |
| `get_capital_flow` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getCapitalFlow` |
| `get_capital_distribution` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getCapitalDistribution` |
| `get_single_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getSinglePromptTemplate` |
| `get_mcp_analysis` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getMCPAnalysis` |
| `fetch_market_news` | — | 外网 | 本地脚本 |
| `fetch_stock_news` | — | 外网 | 本地脚本 |
| `get_report_bot_codes` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getReportBotCodes` |
| `get_stock_daily_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getStockDailyReports` |
| `list_today_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api `POST /getStockDailyReports` |
| `write_execution_log` | — | — | 本地 workspace |
| `recall` | — | — | 本地 chat 存储 |

### 盘前 workflow（16 个，见 `skills/pre_market/manifest.yaml`）

`check_trading_day`, `get_report_bot_codes`, `fetch_market_news`, `fetch_stock_news`, `get_mcp_analysis`, `get_stock_daily_reports`, `list_today_reports`, `get_capital_flow`, `get_capital_distribution`, `get_bot_yesterday_attitude`, `recall_yesterday_summary`, `read_working_state`, `create_pre_market_report`, `save_local_report`, `send_feishu_summary`, `write_execution_log`

---

## 相关文档

- [tool-catalog.md](./tool-catalog.md) — Tool 功能与 MVP 标记
- [clients.md](./clients.md) — HTTP Client 设计
- [geegoo-api-routing.md](../../domains/geegoo-api-routing.md) — GeeGooBot mcp-api 3120 路由规则
