# L2 — 工具目录（完整版）

> **SSOT**：[interface-map.md](../../reference/geegoo-mcp/interface-map.md)  
> 命名：`snake_case` Tool → geegoo mcp :5700 HTTP 路径。  
> **MVP 加粗**。历史 bug 接口见 [clients.md §接口路由说明](./clients.md)。

## 端口与 Client

| Client | 端口 | api_key | 说明 |
|--------|------|---------|------|
| `GeeGooBotClient` / `MarketClient` | **5700** | `sk-...` | 全部 geegoo mcp（MarketClient 为别名） |
| `SignalClient` | **5800** | 无 mcp / 配置 | 信号指标（`signal_base_url`） |

---

## 一、Perception（感知）

### 1.1 市场与账户（geegoo Common + geegoo）

| Tool | API | mcp_token | Phase | MVP | 说明 |
|------|-----|-----------|-------|-----|------|
| **check_trading_day** | 5700 `/checkTradingDay` | 是 | 1 | **✓** | 是否交易日 |
| **get_report_bot_codes** | 5700 `/getReportBotCodes` | 是 | 1 | **✓** | 报告待分析标的，含 bot_id |
| **search_code** | 5700 `/searchCode` | 否 | 4 | | regex + market[]；创建 bot 前必调 |
| **get_position** | 5700 `/getPosition` | 是 | 3/6 | | 富途持仓；SmartTrade sell_only 前置 |
| **get_current_price** | 5700 `/getCurrentPrice` | 否 | 4 | | 最新价 |
| get_ticker | 5700 `/getTicker` | 是 | 3 | | 实时逐笔（盘中） |
| get_broker | 5700 `/getBroker` | 是 | 3 | | 实时经纪队列（盘中） |

### 1.2 新闻与行情（本地脚本 / bundled）

| Tool | 实现 | Phase | MVP | 说明 |
|------|------|-------|-----|------|
| **fetch_market_news** | finance-news US/CN/HK | 1 | **✓** | 市场新闻，内置降级 |
| **fetch_stock_news** | eastmoney + 备选 | 1 | **✓** | 个股新闻 |
| **fetch_global_quote** | global-quotes scripts | 5 | | 免费行情补充 |

---

## 二、Analysis（分析）

### 2.1 技术分析 & Prompt（geegoo AgentAnalyst）

| Tool | API | Phase | MVP | 说明 |
|------|-----|-------|-----|------|
| **get_mcp_analysis** | 5700 `/getMCPAnalysis` | 1 | **✓** | period 必填；name=股票名；读 `analysis_result`；盘前 workflow 常用 5700 |
| **get_single_prompt_template** | 5700 `/getSinglePromptTemplate` | 4 | | type: index/tech/fundamental；可选 period |
| get_tech_prompt_list | 5700 `/getTechPromptList` | 4 | | SKILL 遗留名；优先用上一行 |
| get_stock_daily_reports | 5700 `/getStockDailyReports` | 1 | **✓** | 聚合 pre/intraday/post；**查询报告用此接口** |
| **list_today_reports** | 5700 `/getStockDailyReports` | 1 | **✓** | 幂等检查别名（同日 code） |

### 2.2 资金与态度（geegoo Trading）

| Tool | API | Phase | MVP | 说明 |
|------|-----|-------|-----|------|
| **get_capital_flow** | 5700 `/getCapitalFlow` | 1 | **✓** | 资金流向；`period=DAY`；2026-05-20 已修复 |
| **get_capital_distribution** | 5700 `/getCapitalDistribution` | 1 | **✓** | T-1 资金分布；与上一行同时调用 |
| **get_bot_yesterday_attitude** | 5700 `/getBotYesterdayAttitude` | 1 | **✓** | 404→neutral |
| **get_bot_log_by_type** | 5700 `/getBotLogByType` | 2 | | 盘后：机器人运行日志 |

### 2.3 策略研究（geegoo Strategy + LoopBack）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| **get_index_signals** | 5700 或 5800 `/getIndexSignalForSkill` | 5 | SAR/MACD/BBAND 等指标列表 |
| **get_signal_combinations** | 5700 或 5800 `/getSignalCombinationForSkill` | 5 | 推荐组合信号 |
| **generate_grid_strategy** | 5700 `/generateGridStrategy` | 5 | 网格参数建议 |
| **generate_dca_strategy** | 5700 `/generateDCAStrategy` | 5 | DCA+信号+止盈止损建议 |
| **loopback_strategy** | 5700 `/loopBackStrategy` | 5 | type=dca/grid 回测 |

### 2.4 Prompt 模板管理（geegoo AgentAnalyst — 高级）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| create_competitor_prompt_template | 5700 `/createCompetitorPromptTemplate` | 7 | 用户自建竞品模板 |
| edit_competitor_prompt_template | 5700 `/editCompetitorPromptTemplate` | 7 | |
| delete_competitor_prompt_template | 5700 `/deleteCompetitorPromptTemplate` | 7 | |
| create_etf_prompt_template | 5700 `/createEtfPromptTemplate` | 7 | |
| edit_etf_prompt_template | 5700 `/editEtfPromptTemplate` | 7 | |
| delete_etf_prompt_template | 5700 `/deleteEtfPromptTemplate` | 7 | |

### 2.5 Bot 运行日志（geegoo — 按类型）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| get_dca_bot_log | 5700 `/getDCABotLog` | 6 | log + log_sr + info |
| get_grid_bot_log | 5700 `/getGRIDBotLog` | 6 | |
| get_smart_trade_log | 5700 `/getSmartTradeLog` | 6 | |
| get_dca_reminder_log | 5700 `/getDCAReminderLog` | 6 | |
| get_grid_reminder_log | 5700 `/getGRIDReminderLog` | 6 | |
| get_smart_reminder_log | 5700 `/getSmartReminderLog` | 6 | |

---

## 三、Decision（决策辅助）

| Tool | 实现 | Phase | MVP | 说明 |
|------|------|-------|-----|------|
| **recall_yesterday_summary** | 本地 Episodic | 1 | **✓** | 昨日同股报告摘要 |
| recall_past_attitude | attitude_history.jsonl | 2 | | 态度轨迹 |
| recall_similar_setup | SemanticMemory | 4+ | | 向量检索 |
| compare_daily_reports | get_stock_daily_reports 差分 | 3 | | 盘前 vs 盘中 vs 盘后 |
| **read_working_state** | WorkingMemory | 1 | **✓** | 读结构化进度 |

---

## 四、Action（写入 / 变更）

### 4.1 报告 CRUD（geegoo Market + geegoo 查询）

| Tool | API | Phase | MVP | 说明 |
|------|-----|-------|-----|------|
| **create_pre_market_report** | 5700 `/createPreMarketReport` | 1 | **✓** | 必填见下节 |
| update_pre_market_report | 5700 `/updatePreMarketReport` | 2 | | |
| delete_pre_market_report | 5700 `/deletePreMarketReport` | 2 | | |
| get_pre_market_reports | 5700 `/getPreMarketReports` | 2 | | 2026-05-20 已修复；盘后兜底；按日期聚合仍用 get_stock_daily_reports |
| **create_post_market_report** | 5700 `/createPostMarketReport` | 2 | | 9 字段必填 |
| update_post_market_report | 5700 `/updatePostMarketReport` | 2 | | |
| delete_post_market_report | 5700 `/deletePostMarketReport` | 2 | | |
| get_post_market_reports | 5700 `/getPostMarketReports` | 2 | | 按 code/bot_id/session_date 筛选 |
| create_intraday_report | 5700 `/createIntradayTradeDecisionReport` | 3 | | |
| update_intraday_report | 5700 `/updateIntradayTradeDecisionReport` | 3 | | |
| delete_intraday_report | 5700 `/deleteIntradayTradeDecisionReport` | 3 | | |
| get_intraday_reports | 5700 `/getIntradayTradeDecisionReports` | 3 | | |
| get_daily_reports_unified | 6100 `/reports/daily` | 4 | | reportServer；user_id 鉴权，非 mcp_token |
| **save_local_report** | 本地 FS | 1 | **✓** | 工作区内路径 |
| **send_feishu_summary** | webhook | 1 | 可选 | |

### 4.2 DCA 信号提醒机器人（geegoo DCAReminder）

| Tool | API | Phase | Sandbox |
|------|-----|-------|---------|
| create_dca_reminder | 5700 `/createDCAReminder` | 6 | interactive + **wait_for_human** |
| update_dca_reminder | 5700 `/updateDCAReminder` | 6 | 同上 |
| delete_dca_reminder | 5700 `/deleteDCAReminder` | 6 | 同上 |
| list_dca_reminders | 5700 `/getAllDCAReminders` | 6 | 可读 scheduled |

### 4.3 GRID 网格提醒机器人（geegoo GRIDReminder）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_grid_reminder | 5700 `/createGRIDReminder` | 6 | 默认 frequency=5m，应显式 60m |
| update_grid_reminder | 5700 `/updateGRIDReminder` | 6 | |
| delete_grid_reminder | 5700 `/deleteGRIDReminder` | 6 | |
| list_grid_reminders | 5700 `/getAllGRIDReminders` | 6 | |

### 4.4 Smart 交易提醒机器人（geegoo SmartReminder）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_smart_reminder | 5700 `/createSmartReminder` | 6 | 必填 price + qty；仅 sell_only |
| update_smart_reminder | 5700 `/updateSmartReminder` | 6 | |
| delete_smart_reminder | 5700 `/deleteSmartReminder` | 6 | |
| list_smart_reminders | 5700 `/getAllSmartReminders` | 6 | |

### 4.5 DCA 信号交易机器人（geegoo DCABot）

| Tool | API | Phase |
|------|-----|-------|
| create_dca_bot | 5700 `/createDCABot` | 6 |
| update_dca_bot | 5700 `/updateDCABot` | 6 |
| delete_dca_bot | 5700 `/deleteDCABot` | 6 |
| list_dca_bots | 5700 `/getAllDCABots` | 6 |

### 4.6 GRID 网格交易机器人（geegoo GRIDBot）

| Tool | API | Phase |
|------|-----|-------|
| create_grid_bot | 5700 `/createGRIDBot` | 6 |
| update_grid_bot | 5700 `/updateGRIDBot` | 6 |
| delete_grid_bot | 5700 `/deleteGRIDBot` | 6 |
| list_grid_bots | 5700 `/getAllGRIDBots` | 6 |

### 4.7 SmartTrade 智能交易机器人（geegoo SmartTrade）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_smart_trade | 5700 `/createSmartTrade` | 6 | sell_only 禁手填 price |
| update_smart_trade | 5700 `/updateSmartTrade` | 6 | status 限制改 price/order_size |
| delete_smart_trade | 5700 `/deleteSmartTrade` | 6 | |
| list_smart_trades | 5700 `/getAllSmartTrades` | 6 | |

### 4.8 HDG 对冲交易机器人（geegoo HDGBot）

| Tool | API | Phase | 注意 |
|------|-----|-------|------|
| create_hdg_bot | 5700 `/createHDGBot` | 6 | 需 binding 主 bot_id |
| update_hdg_bot | 5700 `/updateHDGBot` | 6 | 不可改 binding/direction |
| delete_hdg_bot | 5700 `/deleteHDGBot` | 6 | |
| list_hdg_bots | 5700 `/getAllHDGBots` | 6 | |

### 4.9 Bot 开关（Bot 服务经 MCP 转发）

| Tool | API | Phase | 说明 |
|------|-----|-------|------|
| switch_bot | Bot `/switchBot` | 6 | bot_type + bot_id + switch；提醒类常用 |

---

## 五、Meta（元操作）

| Tool | Phase | MVP | 说明 |
|------|-------|-----|------|
| **write_execution_log** | 1 | **✓** | 业务日志 |
| **read_working_state** | 1 | **✓** | |
| update_working_state | 2 | | 一般不由 LLM 直调 |
| spawn_subagent | 2+ | | StockAnalyst / NewsCollector |
| wait_for_human | 6 | | Bot 创建前确认方案 |
| emit_event | 0 | | 调试；正常由 Runtime 发 EventBus |

---

## 六、Skill Pack → Tool 子集

| Skill Pack | 包含 Tool 组 | Phase |
|------------|--------------|-------|
| `pre_market` | §1.1 check/get_user + §1.2 新闻 + §2.1 get_mcp + §2.2 capital/attitude + §4.1 create_pre/save_local + §5 部分 | **1** |
| `post_market` | pre 子集 + get_bot_log + create_post + 3× hourly analysis | 2 |
| `intraday` | get_stock_daily_reports + get_position + create_intraday | 3 |
| `on_demand_analysis` | search_code + get_single_prompt_template + get_mcp_analysis + get_current_price | 4 |
| `strategy` | §2.3 全部 + loopback | 5 |
| `bot_manager` | §4.2–4.9 全部 + §2.5 日志 + search_code + get_position + wait_for_human | 6 |

**Scheduled 模式排除**：§4.2–4.9 所有 `create_*` / `delete_*` / `update_*`（除 report 类 MVP 已控）。

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

| 分类 | 总数 | MVP 实现 |
|------|------|----------|
| Perception | 10 | 5 |
| Analysis | 22 | 7 |
| Decision | 5 | 2 |
| Action | 44 | 3 |
| Meta | 6 | 2 |
| **合计** | **~87** | **~19** |

---

## 九、代码包建议

```text
src/geegoo/tools/
├── perceive.py      # §1
├── analyze.py       # §2.1–2.2
├── analyze_strategy.py  # §2.3 Phase 5
├── analyze_prompts.py # §2.4 Phase 7
├── analyze_logs.py    # §2.5 Phase 6
├── decide.py        # §3
├── act_reports.py   # §4.1
├── act_reminders.py # §4.2–4.4
├── act_bots.py      # §4.5–4.8
├── act_switch.py    # §4.9
└── meta.py          # §5
```

详见 [registry.md](./registry.md)、[clients.md](./clients.md)。
