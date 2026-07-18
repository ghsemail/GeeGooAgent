# GeeGooAgent Tools 运行态总览（2026-07-18）

> **运行态 SSOT**：与代码不一致时以 `internal/tools/` 为准。  
> 相关：[tool-catalog.md](./tool-catalog.md)（设计全集）· [tool-server-mapping.md](./tool-server-mapping.md)（HTTP 路径）· [implementation-status.md](../../implementation-status.md)

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 已实现，生产可用 |
| ⚠️ | 已实现，依赖外部环境或能力降级 |
| 💬 | 接口正常，需先向用户确认参数 |
| 📋 | 已注册，workflow/自动化未接线 |
| ❌ | 未注册或已移除 |

## 后端路由（Tool → 服务 → 端口）

| 后端 | 默认端口 | 主机（生产） | 承载 Tool |
|------|----------|--------------|-----------|
| **GeeGooBot `mcp-api`** | **3120** | 118.195.135.97 | 绝大多数 MCP HTTP；**bespoke** `get_mcp_analysis`/`fetch_*_news` 等亦经此入口（Bot 内转发 analyze/Data） |
| **GeeGooSignal `signal-api`** | **3200** | 146.56.225.252 | `search_code`、`loopback_strategy` |
| **GeeGooSignal `catalog-api`** | **3210** | 146.56.225.252 | `get_index_signals`、`get_signal_combinations` |
| **GeeGooSignal `analyze-api`** | **3230** | 146.56.225.252 | `generate_grid_strategy`、`generate_dca_strategy`（失败可 fallback mcp-api） |
| **GeeGooData `data-api`** | **3300** | A 股 **82.157.97.76**；港/美 **47.80.14.120** | 资金流/行情底层；Agent **不直连**，经 Bot `:3120` 按代码路由 |
| **Agent `agent-runtime`** | **3400** | 119.45.16.112 本机 | ReAct / workflow，非业务 Tool 出口 |
| **Agent 本地** | — | workspace / DuckDuckGo | `web_search`、`save_local_report`、`recall*`、`write_execution_log`；新闻 Bot 不可达时的本地 fallback |

`geegoo doctor` 探测：mcp-api `/health`、`/ready`、`checkTradingDay`；Signal signal/catalog/analyze `/health`；GeeGooData `/health`；**tool 探针**（`search_code`、资金流、富途三类、个股新闻、`get_stock_daily_reports`；`get_mcp_analysis` 跳过因 LLM 慢）。

---

## 统计

| 维度 | 数量 |
|------|------|
| Registry 已注册 | **82** |
| 默认 chat 白名单 | **69**（含 `market` 与 `report_workflow` 共享的 `get_bot_yesterday_attitude`） |
| Bespoke 手写 | **21** |
| HTTP 转发（catalog） | **61** |

---

## 一、Perception（感知）

| Tool | 状态 | 实现 | 来源 / 路径 | 端口 | 备注 |
|------|------|------|-------------|------|------|
| `check_trading_day` | ✅ | bespoke | mcp-api `/checkTradingDay` | 3120 | 需 `mcp_token` |
| `search_code` | ✅ | bespoke | signal-api `/searchCode` → Mongo `stock_db` | 3200 | 不需 `mcp_token`；需 Bearer `signal_api_key` |
| `web_search` | ✅ | bespoke | DuckDuckGo 本地 | — | 股票库无结果时用 |
| `get_current_price` | ✅ | bespoke | mcp-api `/getCurrentPrice` → GeeGooData 行情 | 3120→3300 | **现价快照**；日常查价首选 |
| `get_ticker` | ✅ | HTTP | mcp-api `/getTicker` → `futu_bridge` → 本机 OpenD | 3120 | **盘中逐笔**；非 TradingData；空 payload 重试 1 次 |
| `get_broker` | ✅ | HTTP | mcp-api `/getBroker` → `futu_bridge` → OpenD | 3120 | 同上 |
| `get_position` | ✅ | HTTP | mcp-api `/getPosition` → `futu_bridge` → OpenD | 3120 | 同上；真实空仓仍 skip |
| `fetch_market_news` | ✅ | bespoke | **主路径** Bot `/getMarketNews` → GeeGooData `/v1/news/market`；失败时本地 finance-news / `web_search` | 3120→3300 | 见 [geegoodata-news.md](../../domains/geegoodata-news.md) |
| `fetch_stock_news` | ✅ | bespoke | **主路径** Bot `/getStockNews` → GeeGooData `/v1/news/stock`；失败时本地 / `web_search` | 3120→3300 | 双源仍无 → StatusError |
| `get_report_bot_codes` | ✅ | bespoke | mcp-api `/getReportBotCodes` | 3120 | 盘前待写报告标的 |

---

## 一点五、查价 vs 逐笔（`get_current_price` / `get_ticker`）

| 维度 | `get_current_price` | `get_ticker` |
|------|---------------------|--------------|
| 用途 | 最新价/涨跌幅等**快照** | 盘中**逐笔成交**（time & sales） |
| 调用链 | Agent → Bot `:3120` `/getCurrentPrice` → GeeGooData `:3300` | Agent → Bot `:3120` `/getTicker` → Python `futu_bridge` → **Futu OpenD** `:11111` |
| 是否走 TradingData | 否（GeeGooData） | **否**（TradingData 无逐笔 API，不可替代） |
| 是否需 OpenD | 否 | **是**（Bot 机本机 `127.0.0.1:11111`） |
| 非交易时段 | 通常仍有快照 | 可能为空或仅历史最后几条；doctor 空结果 → `[WARN]` |

`search_code` 为第三条独立链路：Agent → Signal `:3200` `/searchCode` → Mongo `stock_db`（与 Futu / TradingData 无关）。

---

## 二、Analysis（分析）

| Tool | 状态 | 实现 | 来源 / 路径 | 端口 | 备注 |
|------|------|------|-------------|------|------|
| `get_single_prompt_template` | ✅ | bespoke | mcp-api `/getSinglePromptTemplate` | 3120 | `get_mcp_analysis` 前置 |
| `get_mcp_analysis` | 💬/✅ | bespoke | mcp-api `/getMCPAnalysis` → analyze-api LLM | 3120→3230 | 必填 `period`；空结果自动重试 1 次；慢 60–180s |
| `get_capital_flow` | ✅ | bespoke | mcp-api `/getCapitalFlow` → GeeGooData | 3120→3300 | DAY 空自动试 WEEK + 重试；仍无数据 skip |
| `get_capital_distribution` | ✅ | bespoke | mcp-api `/getCapitalDistribution` → GeeGooData | 3120→3300 | 空结果自动重试 1 次 |
| `get_bot_yesterday_attitude` | 💬 | bespoke | mcp-api `/getBotYesterdayAttitude` | 3120 | 必填 `bot_id`；**默认 chat 可用**（`market` 与 `report_workflow` 共享） |
| `get_stock_daily_reports` | 💬 | bespoke | mcp-api `/getStockDailyReports` | 3120 | 建议传 `report_date` |
| `list_today_reports` | ✅ | bespoke | 同上（盘前幂等别名） | 3120 | |
| `list_today_post_market_reports` | ✅ | bespoke | 同上（盘后幂等别名） | 3120 | `post_market` workflow |
| `get_index_signals` | ✅ | HTTP | catalog-api `/getIndexSignalForSkill` | 3210 | DCA 单指标 |
| `get_signal_combinations` | ✅ | HTTP | catalog-api `/getSignalCombinationForSkill` | 3210 | DCA 组合信号 |
| `generate_grid_strategy` | 💬/✅ | HTTP | analyze-api `/generateGridStrategy` | 3230 | 空 `param` → skip（重试 1 次）；有 grid 字段即 OK（`suitable=false` 亦可） |
| `generate_dca_strategy` | 💬/✅ | HTTP | analyze-api `/generateDCAStrategy` | 3230 | 空 `signal.buy_signal` → skip（重试 1 次）；可 fallback 3120 |
| `loopback_strategy` | 💬/✅ | HTTP | signal-api `/loopBackStrategy` | 3200 | 需先 `generate_*` 拿参数 |
| `get_bot_log_by_type` | ✅ | HTTP | mcp-api `/getBotLogByType` | 3120 | 必填 `type` + `bot_id` |

### Prompt 模板 CRUD（6）

| Tool | 状态 | 路径 | 端口 |
|------|------|------|------|
| `create/edit/delete_*_competitor_prompt_template` | ✅ | mcp-api 对应路径 | 3120 | `/toolsets prompt_template`；chat 写操作需确认 |
| `create/edit/delete_*_etf_prompt_template` | ✅ | mcp-api 对应路径 | 3120 | 同上 |

---

## 三、Decision（决策辅助）

| Tool | 状态 | 实现 | 来源 | 端口 | 备注 |
|------|------|------|------|------|------|
| `recall` | ✅ | bespoke | SQLite FTS 跨会话 | — | |
| `recall_yesterday_summary` | ✅ | bespoke | 本地 `reports/<date>/<code>-premarket.md`；fallback MCP 向前 5 天 | — | 均无则 skip |
| `read_working_state` | ✅ | bespoke | WorkingMemory 状态库 | — | workflow 内 |

---

## 四、Action — 报告 CRUD

| Tool | 状态 | 路径 | 端口 | 默认 chat |
|------|------|------|------|-----------|
| `create_pre_market_report` | 💬 | `/createPreMarketReport` | 3120 | 🔒 workflow |
| `update/delete/get_pre_market_reports` | ✅ | 对应路径 | 3120 | query |
| `create/update/delete/get_intraday_reports` | ✅ | Intraday 路径 | 3120 | query / workflow |
| `create/update/delete/get_post_market_reports` | ✅ | PostMarket 路径 | 3120 | query / workflow |
| `save_local_report` | ✅ | 本地 workspace | — | 🔒 workflow |

---

## 五、Action — Bot / Reminder（各 5：create/update/delete/list/get_*_log）

| 族 | 状态 | 路径前缀 | 端口 | 备注 |
|----|------|----------|------|------|
| `dca_bot` / `grid_bot` / `smart_trade` / `hdg_bot` | 💬/✅ | `/create*Bot` 等 | 3120 | 创建后 **无 scheduler 自动跑** ⚠️ |
| `dca_reminder` / `grid_reminder` / `smart_reminder` | 💬/✅ | `/create*Reminder` 等 | 3120 | 写操作需用户确认 |

---

## 六、Meta

| Tool | 状态 | 实现 | 备注 |
|------|------|------|------|
| `write_execution_log` | ✅ | 本地 `execution-log.md` | workflow 用 |

---

## 七、Skill / Workflow 与 Tool 关系

| Skill | 状态 | CLI | manifest 白名单 |
|-------|------|-----|-----------------|
| `pre_market` | ✅ | `geegoo run pre_market` | 15 Tool（无 `send_feishu`） |
| `intraday` | ✅ | `geegoo run intraday --code … --bot-id …` | 9 Tool：持仓、报告、资金、分析、现价 |
| `post_market` | ✅ | `geegoo run post_market` | 10 Tool：3× hourly 分析 + bot log + post 报告 |

manifest 路径：`skills/<skill>/manifest.yaml`。Skill 文档 → [L5 skills](../L5-application/skills.md)。

---

## 八、需引导用户选择（💬）

| Tool | Agent 应先做什么 |
|------|------------------|
| `generate_dca_strategy` | 问单指标/组合 → `get_index_signals` / `get_signal_combinations` → 用户选 `signal_id` |
| `loopback_strategy` | grid：先 `generate_grid_strategy`；dca：确认 `signal` + `sl_tp` |
| `get_mcp_analysis` | 确认 `period`（及 `prompt_id`）；可用 `get_single_prompt_template` |
| `get_bot_yesterday_attitude` | 列机器人，让用户指定 `bot_id` |
| `get_stock_daily_reports` | 确认 `report_date`（`YYYY-MM-DD`） |
| `create_*_report` / Bot 写操作 | 确认 code、stock_name 等；默认 chat 不含 🔒 workflow 工具 |

---

## 九、韧性策略速查（`internal/tools/resilience.go`）

| Tool | 策略 |
|------|------|
| `get_position` / `get_ticker` / `get_broker` | HTTP 空 payload → 等 2s 重试 1 次 |
| `generate_grid_strategy` / `generate_dca_strategy` | HTTP 空 payload → 等 2s 重试 1 次；仍无可用字段 → skip |
| `get_capital_flow` | DAY 空试 WEEK；整轮重试 1 次 |
| `get_capital_distribution` | 空分布重试 1 次 |
| `get_mcp_analysis` | 空 `analysis_result` 重试 1 次 |
| `fetch_stock_news` | Bot→Data 无标题 → 本地 finance-news → `web_search`；仍无 → `StatusError` |
| `fetch_market_news` | 同上 |
| `recall_yesterday_summary` | 本地无文件 → MCP 报告向前查 5 天 |

---

## 十、常踩坑速查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| 资金类 skip | Bot→CN 节点防火墙 / Token / 标的无成交 | `verify_e2e_capital.py`；查 Bot `.env` 路由 |
| `search_code` doctor FAIL | 曾误截断响应体；或 `signal_api_key` 未配 | 用 `mcp.Client.SearchCode` 探针；确认 `:3200` Bearer |
| 富途三类 skip / 500 | OpenD 未起、非交易时段、或 bridge stdout 混日志 | 查 Bot `mcp-api.out`；查价改用 `get_current_price` |
| 新闻 `StatusError` | 双源均无标题 | 检查网络；`fetch_market_news` / `fetch_stock_news` 行为一致 |
| generate 503 | analyze-api 或 LLM 未配 | `curl :3230/health` |
| Bot 创建不跑 | GeeGooBot 无 scheduler | 架构缺口，非 Agent bug |

---

## 十一、默认 chat Toolset

| ID | 默认 chat | 工具数 | 说明 |
|----|-----------|--------|------|
| `market` | ✅ | 18 | 行情、分析、新闻（含 `get_bot_yesterday_attitude`） |
| `strategy` | ✅ | 3 | generate + loopback |
| `bot_manager` | ✅ | 20 | 交易 Bot CRUD + log |
| `reminder_manager` | ✅ | 15 | 提醒 Bot |
| `report_query` | ✅ | **13** | 报告查询 |
| `report_workflow` | 🔒 | **8** | 盘前/盘后 workflow（含 `list_today_post_market_reports`） |
| `prompt_template` | 🔒 | **6** | 竞品/ETF Prompt 模板 CRUD |

切换：`/toolsets market,strategy` · `/toolsets default` · `/toolsets prompt_template`（高级）

> **共享 tool**：`get_bot_yesterday_attitude` 同时在 `market` 与 `report_workflow`；默认 chat **保留**（仅 workflow 独占 tool 从默认 chat 排除）。

---

## 十二、树形总览（82 registered）

```text
GeeGooAgent Tools
├─ market [toolset]
│  ├─ search_code, web_search, check_trading_day, get_current_price     ✅
│  ├─ get_ticker, get_broker, get_position                              ✅ futu_bridge + 空 payload 重试
│  ├─ get_capital_flow, get_capital_distribution                        ✅ Bot→GeeGooData 分节点
│  ├─ get_bot_yesterday_attitude                                        💬 需 bot_id
│  ├─ get_index_signals, get_signal_combinations                        ✅ :3210
│  ├─ get_single_prompt_template, get_mcp_analysis                     💬 需 period
│  ├─ fetch_market_news, fetch_stock_news                               ✅ Bot→GeeGooData；本地 fallback
│  ├─ get_bot_log_by_type, recall                                       ✅
├─ strategy
│  ├─ generate_grid_strategy, generate_dca_strategy                     💬/✅ :3230
│  └─ loopback_strategy                                                 💬 需 generate_* 参数链
├─ bot_manager / reminder_manager（各 5× CRUD + log）
├─ report_query（盘前/盘中/盘后 CRUD + 聚合 + list_today_reports）
├─ report_workflow 🔒（create_pre_market, list_today_post_market_reports, save_local, …）
└─ prompt_template 🔒（竞品/ETF 模板 CRUD×6；读模板用 market 内 get_single_prompt_template）
```

实现层：**Agent/Go** · **Bot/Go** :3120 · **Signal/Go** :3200/3210 · **Analyze/Go** :3230 · **Data/Go** :3300 · **本地**

---

## 十三、对话场景推荐

| 目标 | 推荐 | 避免 |
|------|------|------|
| 查价 | `search_code` → `get_current_price` | 用 `get_ticker` 当现价用（链路更重、依赖 OpenD） |
| 盘中逐笔 | `get_ticker`（交易时段） | 期望走 GeeGooData / TradingData（无此能力） |
| 技术分析 | `get_single_prompt_template` → `get_mcp_analysis` | 缺 `period` |
| DCA 方案 | 先选信号 → `generate_dca_strategy` → 可选 `loopback_strategy` | 未选 `signal_id` 就 generate |
| 新闻 | `fetch_stock_news` / `fetch_market_news`；库无结果用 `web_search` | — |
| 盘前写报告 | `/toolsets report_workflow` 或 `geegoo run pre_market` | 默认 chat 白名单 |
| 盘中决策 | `geegoo run intraday --code …` | 默认 chat |
| 盘后总结 | `geegoo run post_market` | — |

---

## 十四、维护

新增 Tool 后同步更新：**本文件** + [tool-catalog.md](./tool-catalog.md) + `catalog/catalog.go` + [interface-map.md](../../../reference/geegoo-mcp/interface-map.md)（新 HTTP 时）。

新闻迁入 GeeGooData 已完成（主路径 Bot→Data）；维护时同步 [geegoodata-news.md](../../domains/geegoodata-news.md)、GeeGooData `docs/NEWS.md`、Bot `interface-map`。

核对命令：

```bash
go test ./internal/tools/... -count=1
python scripts/verify_e2e_news.py    # Bot 新闻路由
```
