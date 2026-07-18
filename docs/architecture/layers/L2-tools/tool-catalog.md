# L2 — 工具目录（设计全集）

> **运行态 SSOT**（已注册 / 可用性 / 端口 / 后端）：[tools-status.md](./tools-status.md)  
> **MCP HTTP SSOT**：[interface-map.md](../../../reference/geegoo-mcp/interface-map.md)  
> 命名：`snake_case` Tool → HTTP 路径；**MVP 加粗**。  
> 表内无「注册」列时，默认 ✅ 已注册；**运行态**见 [tools-status.md](./tools-status.md)。

## 端口与 Client

| Client | 端口 | api_key | 说明 |
|--------|------|---------|------|
| `GeeGooBotClient` / `MarketClient` | **3120** | `sk-...` | 全部 GeeGooBot mcp-api（MarketClient 为别名） |
| `SignalClient` | **3200** / **3210** / **3230** | Bearer | signal-api 搜索/回测；catalog-api 指标；analyze-api 策略生成 |

---

## 一、Perception（感知）

### 1.1 市场与账户（geegoo Common + geegoo）

| Tool | API | mcp_token | Phase | MVP | 说明 |
|------|-----|-----------|-------|-----|------|
| **check_trading_day** | 3120 `/checkTradingDay` | 是 | 1 | **✓** | 是否交易日 |
| **get_report_bot_codes** | 3120 `/getReportBotCodes` | 是 | 1 | **✓** | 报告待分析标的，含 bot_id |
| **search_code** | 3200 `/searchCode` | 否 | 4 | | bespoke；Signal signal-api |
| **web_search** | Agent 本地 DuckDuckGo | — | 4 | | 已注册；非 MCP |
| **get_position** | 3120 `/getPosition` | 是 | 3/6 | | 富途 OpenD；空 payload 重试；真实空仓 skip |
| **get_current_price** | 3120 `/getCurrentPrice` | 是* | 4 | | bespoke → Data |
| get_ticker | 3120 `/getTicker` | 是 | 3 | | 富途 OpenD；空 payload 重试 |
| get_broker | 3120 `/getBroker` | 是 | 3 | | 富途 OpenD；空 payload 重试 |

### 1.2 新闻与行情（本地脚本 / bundled）

| Tool | 实现 | Phase | MVP | 说明 |
|------|------|-------|-----|------|
| **fetch_market_news** | finance-news US/CN/HK | 1 | **✓** | 市场新闻，内置降级 |
| **fetch_stock_news** | eastmoney + Go 多源 + web_search | 1 | **✓** | 双源仍无则 StatusError |

---

## 二、Analysis（分析）

### 2.1 技术分析 & Prompt（geegoo AgentAnalyst）

| Tool | API | Phase | MVP | 说明 |
|------|-----|-------|-----|------|
| **get_mcp_analysis** | 3120 `/getMCPAnalysis` | 1 | **✓** | period 必填；name=股票名；读 `analysis_result`；盘前 workflow 常用 3120 |
| **get_single_prompt_template** | 3120 `/getSinglePromptTemplate` | 4 | | type: index/tech/fundamental；可选 period |
| get_stock_daily_reports | 3120 `/getStockDailyReports` | 1 | **✓** | 聚合 pre/intraday/post；**查询报告用此接口** |
| **list_today_reports** | 3120 `/getStockDailyReports` | 1 | **✓** | 盘前幂等检查别名（同日 code） |
| **list_today_post_market_reports** | 3120 `/getStockDailyReports` | 2 | | 盘后幂等检查别名 |

### 2.2 资金与态度（geegoo Trading）

| Tool | API | Phase | MVP | 说明 |
|------|-----|-------|-----|------|
| **get_capital_flow** | 3120 `/getCapitalFlow` | 1 | **✓** | 资金流向；`period=DAY`；2026-05-20 已修复 |
| **get_capital_distribution** | 3120 `/getCapitalDistribution` | 1 | **✓** | T-1 资金分布；与上一行同时调用 |
| **get_bot_yesterday_attitude** | 3120 `/getBotYesterdayAttitude` | 1 | **✓** | 404→neutral |
| **get_bot_log_by_type** | 3120 `/getBotLogByType` | 2 | | 盘后：机器人运行日志 |

### 2.3 策略研究（geegoo Strategy + LoopBack）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| **get_index_signals** | 3210 `/getIndexSignalForSkill` | 5 | SAR/MACD/BBAND 等指标列表 |
| **get_signal_combinations** | 3210 `/getSignalCombinationForSkill` | 5 | 推荐组合信号 |
| **generate_grid_strategy** | 3230 `/generateGridStrategy`（fallback 3120） | 5 | 网格参数建议 |
| **generate_dca_strategy** | 3230 `/generateDCAStrategy`（fallback 3120） | 5 | DCA+信号+止盈止损；需先选 signal_id |
| **loopback_strategy** | 3200 `/loopBackStrategy` | 5 | Signal 原生回测 |

### 2.4 Prompt 模板管理（geegoo AgentAnalyst — 高级）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| create_competitor_prompt_template | 3120 `/createCompetitorPromptTemplate` | 7 | 用户自建竞品模板 |
| edit_competitor_prompt_template | 3120 `/editCompetitorPromptTemplate` | 7 | |
| delete_competitor_prompt_template | 3120 `/deleteCompetitorPromptTemplate` | 7 | |
| create_etf_prompt_template | 3120 `/createEtfPromptTemplate` | 7 | |
| edit_etf_prompt_template | 3120 `/editEtfPromptTemplate` | 7 | |
| delete_etf_prompt_template | 3120 `/deleteEtfPromptTemplate` | 7 | |

### 2.5 Bot 运行日志（geegoo — 按类型）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| get_dca_bot_log | 3120 `/getDCABotLog` | 6 | log + log_sr + info |
| get_grid_bot_log | 3120 `/getGRIDBotLog` | 6 | |
| get_smart_trade_log | 3120 `/getSmartTradeLog` | 6 | |
| get_dca_reminder_log | 3120 `/getDCAReminderLog` | 6 | |
| get_grid_reminder_log | 3120 `/getGRIDReminderLog` | 6 | |
| get_smart_reminder_log | 3120 `/getSmartReminderLog` | 6 | |
| get_hdg_bot_log | 3120 `/getHDGBotLog` | 6 | HDG 对冲 Bot 日志 |

---

## 三、Decision（决策辅助）

| Tool | 实现 | Phase | MVP | 说明 |
|------|------|-------|-----|------|
| **recall** | chatsession FTS | 4 | | 跨会话检索；已注册 |
| **recall_yesterday_summary** | 本地 Episodic | 1 | **✓** | 无本地/MCP 报告时 skip；见 [tools-status.md](./tools-status.md) |
| **read_working_state** | WorkingMemory | 1 | **✓** | 读结构化进度 |

---

## 四、Action（写入 / 变更）

### 4.1 报告 CRUD（geegoo Market + geegoo 查询）

| Tool | API | Phase | MVP | 说明 |
|------|-----|-------|-----|------|
| **create_pre_market_report** | 3120 `/createPreMarketReport` | 1 | **✓** | 必填见下节 |
| update_pre_market_report | 3120 `/updatePreMarketReport` | 2 | | |
| delete_pre_market_report | 3120 `/deletePreMarketReport` | 2 | | |
| get_pre_market_reports | 3120 `/getPreMarketReports` | 2 | | 2026-05-20 已修复；盘后兜底；按日期聚合仍用 get_stock_daily_reports |
| **create_post_market_report** | 3120 `/createPostMarketReport` | 2 | | 9 字段必填 |
| update_post_market_report | 3120 `/updatePostMarketReport` | 2 | | |
| delete_post_market_report | 3120 `/deletePostMarketReport` | 2 | | |
| get_post_market_reports | 3120 `/getPostMarketReports` | 2 | | 按 code/bot_id/session_date 筛选 |
| create_intraday_report | 3120 `/createIntradayTradeDecisionReport` | 3 | | |
| update_intraday_report | 3120 `/updateIntradayTradeDecisionReport` | 3 | | |
| delete_intraday_report | 3120 `/deleteIntradayTradeDecisionReport` | 3 | | |
| get_intraday_reports | 3120 `/getIntradayTradeDecisionReports` | 3 | | |
| **save_local_report** | 本地 FS | 1 | **✓** | 工作区内路径 |
| ~~send_feishu_summary~~ | — | — | ❌ 已移除 | 待 GeeGooBot Notify Gateway |

### 4.2 DCA 信号提醒机器人（geegoo DCAReminder）

| Tool | API | Phase | Sandbox |
|------|-----|-------|---------|
| create_dca_reminder | 3120 `/createDCAReminder` | 6 | interactive + ApprovalGate |
| update_dca_reminder | 3120 `/updateDCAReminder` | 6 | 同上 |
| delete_dca_reminder | 3120 `/deleteDCAReminder` | 6 | 同上 |
| list_dca_reminders | 3120 `/getAllDCAReminders` | 6 | 可读 scheduled |

### 4.3 GRID 网格提醒机器人（geegoo GRIDReminder）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_grid_reminder | 3120 `/createGRIDReminder` | 6 | 默认 frequency=5m，应显式 60m |
| update_grid_reminder | 3120 `/updateGRIDReminder` | 6 | |
| delete_grid_reminder | 3120 `/deleteGRIDReminder` | 6 | |
| list_grid_reminders | 3120 `/getAllGRIDReminders` | 6 | |

### 4.4 Smart 交易提醒机器人（geegoo SmartReminder）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_smart_reminder | 3120 `/createSmartReminder` | 6 | 必填 price + qty；仅 sell_only |
| update_smart_reminder | 3120 `/updateSmartReminder` | 6 | |
| delete_smart_reminder | 3120 `/deleteSmartReminder` | 6 | |
| list_smart_reminders | 3120 `/getAllSmartReminders` | 6 | |

### 4.5 DCA 信号交易机器人（geegoo DCABot）

| Tool | API | Phase |
|------|-----|-------|
| create_dca_bot | 3120 `/createDCABot` | 6 |
| update_dca_bot | 3120 `/updateDCABot` | 6 |
| delete_dca_bot | 3120 `/deleteDCABot` | 6 |
| list_dca_bots | 3120 `/getAllDCABots` | 6 |

### 4.6 GRID 网格交易机器人（geegoo GRIDBot）

| Tool | API | Phase |
|------|-----|-------|
| create_grid_bot | 3120 `/createGRIDBot` | 6 |
| update_grid_bot | 3120 `/updateGRIDBot` | 6 |
| delete_grid_bot | 3120 `/deleteGRIDBot` | 6 |
| list_grid_bots | 3120 `/getAllGRIDBots` | 6 |

### 4.7 SmartTrade 智能交易机器人（geegoo SmartTrade）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_smart_trade | 3120 `/createSmartTrade` | 6 | sell_only 禁手填 price |
| update_smart_trade | 3120 `/updateSmartTrade` | 6 | status 限制改 price/order_size |
| delete_smart_trade | 3120 `/deleteSmartTrade` | 6 | |
| list_smart_trades | 3120 `/getAllSmartTrades` | 6 | |

### 4.8 HDG 对冲交易机器人（geegoo HDGBot）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_hdg_bot | 3120 `/createHDGBot` | 6 | 需 binding 主 bot_id |
| update_hdg_bot | 3120 `/updateHDGBot` | 6 | 不可改 binding/direction |
| delete_hdg_bot | 3120 `/deleteHDGBot` | 6 | |
| list_hdg_bots | 3120 `/getAllHDGBots` | 6 | |

## 五、Meta（元操作）

| Tool | Phase | MVP | 说明 |
|------|-------|-----|------|
| **write_execution_log** | 1 | **✓** | 业务日志 |

---

## 六、Skill Pack → Tool 子集

| Skill Pack | 包含 Tool 组 | Phase | manifest |
|------------|--------------|-------|----------|
| `pre_market` | §1.1–1.2 + §2.1–2.2 + §4.1 create_pre/save_local + §5 Meta | **1** | `skills/pre_market/manifest.yaml`（15） |
| `post_market` | check + bot codes + list_today_post_market + 3× hourly analysis + bot log + post 报告 | 2 | `skills/post_market/manifest.yaml`（10） |
| `intraday` | get_position + daily reports + capital + hourly analysis + intraday 报告 | 3 | `skills/intraday/manifest.yaml`（9） |
| `on_demand_analysis` | search_code + get_single_prompt_template + get_mcp_analysis + get_current_price | 4 | — |
| `strategy` | §2.3 全部 + loopback | 5 | — |
| `bot_manager` | §4.2–4.8 全部 + §2.5 日志 + search_code + get_position | 6 | — |

**Scheduled 模式排除**：§4.2–4.8 所有 `create_*` / `delete_*` / `update_*`（除 report 类 MVP 已控）。

---

## 七、关键校验

### create_pre_market_report（MVP 强制）

`mcp_token`, `code`, `stock_name`, `bot_id`, `bot_name`, `bot_type`, `result`, `confidence`, `reason`, `suggestion`, `report`

枚举：`result` ∈ long/short/neutral；`suggestion` ∈ buy/sell/hold；`confidence` ∈ high/medium/low

### get_mcp_analysis

- `period` 必填：`hourly` | `weekly` | `daily` | …（非 `hour`）
- `name` = 股票名称（如「腾讯控股」），**不是** Prompt 名

### attitude → result

| get_bot_yesterday_attitude | create_pre_market_report |
|--------------------------|--------------------------|
| bullish | long |
| bearish | short |
| neutral | neutral |

---

## 八、工具数量统计

| 分类 | 已注册 (2026-07) |
|------|------------------|
| Perception | 10 |
| Analysis | 22 |
| Decision | 3 |
| Action | 42 |
| Meta | 1 |
| **合计** | **82** |

运行态明细 → [tools-status.md](./tools-status.md)

---

## 九、代码包（Go）

```text
internal/tools/
├── registry.go, bootstrap.go, bespoke.go, resilience.go
├── catalog/catalog.go    # HTTP 规格
├── toolset.go, domains.go
└── approval.go, contract.go, httpbackend.go
```

详见 [registry.md](./registry.md)、[toolsets.md](./toolsets.md)、[clients.md](./clients.md)。
