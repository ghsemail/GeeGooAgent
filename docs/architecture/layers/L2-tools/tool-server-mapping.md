# Tool 服务器与接口对照表

> 维护：Tool 注册以 `internal/tools/catalog/catalog.go` 与 `bespoke.go` 为准；改 API 时同步本表与 [interface-map.md](../../../reference/geegoo-mcp/interface-map.md)。  
> **运行态可用性**：[tools-status.md](./tools-status.md)

## 服务器别名（GeeGoo Go 3xxx）

| 别名 | IP | 端口 | 说明 |
|------|-----|------|------|
| **GeeGooBot mcp-api** | `118.195.135.97` | **3120** | MCP 契约 API（GeeGooAgent 主入口） |
| **GeeGooSignal signal-api** | `146.56.225.252` | **3200** | 标的搜索、回测等 |
| **GeeGooSignal catalog-api** | `146.56.225.252` | **3210** | 指标信号、queryModel |
| **GeeGooSignal analyze-api** | `146.56.225.252` | **3230** | MCP 分析、策略生成（Agent 可直连） |
| **GeeGooData data-api（港/美）** | `47.80.14.120` | **3300** | 港美股行情、资金底层 |
| **GeeGooData data-api（A 股）** | `82.157.97.76` | **3300** | A 股行情、资金底层（经 Bot 按 code 路由） |

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
| `data_base_url` | `http://47.80.14.120:3300` | GeeGooData 默认（港/美）；A 股由 Bot 路由至 `82.157.97.76:3300` |

参考：`config.production.example.json`

## 图例

| 列 | 含义 |
|----|------|
| **直连服务器** | Agent HTTP 直接访问的服务器 |
| **间接服务器** | GeeGooBot mcp-api 服务端内部再调用的下游（Agent 不直连） |
| **接口服务** | GeeGooBot mcp-api 上的具体 API 进程与端口 |
| **接口方法** | HTTP 方法与路径（均为 `POST`） |

特殊值：

- **—** — 不依赖 GeeGoo Go 后端（本地 workspace / 外网）
- **外网 HTTPS** — 新闻 RSS、DuckDuckGo 等公网出站

---

## 按服务器汇总

| 服务器 | 涉及 Tool 数 | 说明 |
|--------|-------------|------|
| **GeeGooBot mcp-api** | 74 | 绝大多数 HTTP Tool 入口 |
| **GeeGooSignal**（Agent 可直连） | 5 | search、catalog、analyze、loopback |
| **GeeGooData**（间接，经 Bot） | 2+ | 现价、资金、分布 |
| **无（本地/外网）** | 7 | workspace、新闻、web_search、recall |

---

## 间接依赖速查

| 下游 | 触发 Tool | Agent 入口 |
|------|----------|------------|
| **GeeGooData** `:3300` | `get_current_price`, `get_capital_flow`, `get_capital_distribution` | Bot `POST` 对应路径（按 code 路由 CN/HK/US 节点） |
| **富途 OpenD** | `get_position`, `get_ticker`, `get_broker` | Bot `POST /getPosition` 等 |
| **GeeGooSignal catalog** `:3210` | `get_index_signals`, `get_signal_combinations` | Agent 直连 catalog-api |
| **GeeGooSignal analyze** `:3230` | `generate_grid_strategy`, `generate_dca_strategy` | Agent 直连 analyze-api（失败 fallback 3120） |
| **GeeGooSignal signal** `:3200` | `search_code`, `loopback_strategy` | Agent 直连 signal-api |

---

## 一、感知 Perception

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `check_trading_day` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /checkTradingDay` |
| `search_code` | GeeGooSignal signal-api | — | GeeGooSignal signal-api:3200 | `POST /searchCode` |
| `get_current_price` | GeeGooBot mcp-api | GeeGooData | GeeGooBot mcp-api:3120 | `POST /getCurrentPrice` |
| `get_ticker` | GeeGooBot mcp-api | 富途 OpenD | GeeGooBot mcp-api:3120 | `POST /getTicker` |
| `get_position` | GeeGooBot mcp-api | 富途 OpenD | GeeGooBot mcp-api:3120 | `POST /getPosition` |
| `get_broker` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getBroker` |
| `get_report_bot_codes` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getReportBotCodes` |
| `fetch_market_news` | GeeGooBot :3120 | GeeGooData :3300 | Bot→Data 多源聚合 + Agent 本地回退 |
| `fetch_stock_news` | GeeGooBot :3120 | GeeGooData :3300 | Bot→Data 多源聚合 + web_search |

---

## 二、分析 Analysis

| Tool | 直连服务器 | 间接服务器 | 接口服务 | 接口方法 |
|------|-----------|-----------|----------|----------|
| `get_mcp_analysis` | GeeGooBot mcp-api | analyze-api | GeeGooBot mcp-api:3120 | `POST /getMCPAnalysis`（内部可转 3230） |
| `get_capital_flow` | GeeGooBot mcp-api | GeeGooData | GeeGooBot mcp-api:3120 | `POST /getCapitalFlow` |
| `get_capital_distribution` | GeeGooBot mcp-api | GeeGooData | GeeGooBot mcp-api:3120 | `POST /getCapitalDistribution` |
| `get_bot_yesterday_attitude` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getBotYesterdayAttitude` |
| `get_bot_log_by_type` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getBotLogByType` |
| `get_stock_daily_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getStockDailyReports` |
| `list_today_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getStockDailyReports`（盘前幂等别名） |
| `list_today_post_market_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getStockDailyReports`（盘后幂等别名） |
| `get_pre_market_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getPreMarketReports` |
| `get_intraday_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getIntradayTradeDecisionReports` |
| `get_post_market_reports` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getPostMarketReports` |
| `get_index_signals` | GeeGooSignal catalog-api | — | catalog-api:3210 | `POST /getIndexSignalForSkill` |
| `get_signal_combinations` | GeeGooSignal catalog-api | — | catalog-api:3210 | `POST /getSignalCombinationForSkill` |
| `get_single_prompt_template` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120 | `POST /getSinglePromptTemplate` |
| `generate_grid_strategy` | GeeGooSignal analyze-api | — | analyze-api:3230 | `POST /generateGridStrategy`（fallback 3120） |
| `generate_dca_strategy` | GeeGooSignal analyze-api | — | analyze-api:3230 | `POST /generateDCAStrategy`（fallback 3120） |
| `loopback_strategy` | GeeGooSignal signal-api | — | signal-api:3200 | `POST /loopBackStrategy` |
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

### `geegoo chat`（~73 个，按 toolset 白名单）

详见 [tools-status.md](./tools-status.md) §十一 Toolset。核心：`market` + `strategy` + `bot_manager` + `reminder_manager` + `report_query`。

### 盘前 workflow（15 个，见 `skills/pre_market/manifest.yaml`）

`check_trading_day`, `get_report_bot_codes`, `fetch_market_news`, `fetch_stock_news`, `get_mcp_analysis`, `get_stock_daily_reports`, `list_today_reports`, `get_capital_flow`, `get_capital_distribution`, `get_bot_yesterday_attitude`, `recall_yesterday_summary`, `read_working_state`, `create_pre_market_report`, `save_local_report`, `write_execution_log`

### 盘中 / 盘后

见 `skills/intraday/manifest.yaml`（9 Tool）、`skills/post_market/manifest.yaml`（10 Tool）。

---

## 相关文档

- [tools-status.md](./tools-status.md) — 运行态 SSOT
- [tool-catalog.md](./tool-catalog.md) — Tool 功能与 MVP 标记
- [clients.md](./clients.md) — HTTP Client 设计
- [geegoo-api-routing.md](../../domains/geegoo-api-routing.md) — GeeGooBot mcp-api 3120 路由规则
