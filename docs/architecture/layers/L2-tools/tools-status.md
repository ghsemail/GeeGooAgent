# GeeGooAgent Tools 运行态总览（2026-07-17）

> **SSOT**：与代码不一致时以 `internal/tools/` 为准。  
> 相关：[tools-tree.md](./tools-tree.md) · [tool-catalog.md](./tool-catalog.md) · [implementation-status.md](../../implementation-status.md)

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
| **GeeGooBot `mcp-api`** | **3120** | 118.195.135.97 | 绝大多数 MCP HTTP（行情、报告、Bot CRUD、资金流经 Bot 转发） |
| **GeeGooSignal `signal-api`** | **3200** | 146.56.225.252 | `search_code`、`loopback_strategy` |
| **GeeGooSignal `catalog-api`** | **3210** | 146.56.225.252 | `get_index_signals`、`get_signal_combinations` |
| **GeeGooSignal `analyze-api`** | **3230** | 146.56.225.252 | `generate_grid_strategy`、`generate_dca_strategy`（失败可 fallback mcp-api） |
| **GeeGooData `data-api`** | **3300** | A 股 **82.157.97.76**；港/美 **47.80.14.120** | 资金流/行情底层；Agent **不直连**，经 Bot `:3120` 按代码路由 |
| **Agent `agent-runtime`** | **3400** | 119.45.16.112 本机 | ReAct / workflow，非业务 Tool 出口 |
| **Agent 本地** | — | workspace / DuckDuckGo | `fetch_*_news`、`web_search`、`save_local_report`、`recall*`、`write_execution_log` |

`geegoo doctor` 探测：mcp-api `/health`、`/ready`、`checkTradingDay`；Signal signal/catalog/analyze `/health`；GeeGooData `/health`；**tool 探针**（`search_code`、资金流、富途三类、个股新闻、`get_stock_daily_reports`；`get_mcp_analysis` 跳过因 LLM 慢）。

---

## 统计

| 维度 | 数量 |
|------|------|
| Registry 已注册 | **≥80** |
| 默认 chat 白名单 | **~73** |
| Bespoke 手写 | **21** |
| HTTP 转发（catalog） | **60** |

---

## 一、Perception（感知）

| Tool | 状态 | 实现 | 来源 / 路径 | 端口 | 备注 |
|------|------|------|-------------|------|------|
| `check_trading_day` | ✅ | bespoke | mcp-api `/checkTradingDay` | 3120 | 需 `mcp_token` |
| `search_code` | ✅ | bespoke | signal-api `/searchCode` | 3200 | 不需 `mcp_token` |
| `web_search` | ✅ | bespoke | DuckDuckGo 本地 | — | 股票库无结果时用 |
| `get_current_price` | ✅ | bespoke | mcp-api `/getCurrentPrice` | 3120 | 现价快照 |
| `get_ticker` | ✅ | HTTP | mcp-api `/getTicker` | 3120 | 空 payload 自动重试 1 次（2s）；非交易时段仍可能 skip |
| `get_broker` | ✅ | HTTP | mcp-api `/getBroker` | 3120 | 同上 |
| `get_position` | ✅ | HTTP | mcp-api `/getPosition` | 3120 | 同上；真实空仓仍 skip |
| `fetch_market_news` | ✅ | bespoke | 本地 Go RSS/新浪 + Python 回退 | — | US/CN/HK |
| `fetch_stock_news` | ✅ | bespoke | finance-news + Go 多源 + `web_search` | — | 港股：Yahoo `.HK` → ADR → 新浪；A 股：东财公告 → 新浪；双源仍无则 `StatusError` |
| `get_report_bot_codes` | ✅ | bespoke | mcp-api `/getReportBotCodes` | 3120 | 盘前待写报告标的 |

---

## 二、Analysis（分析）

| Tool | 状态 | 实现 | 来源 / 路径 | 端口 | 备注 |
|------|------|------|-------------|------|------|
| `get_single_prompt_template` | ✅ | bespoke | mcp-api `/getSinglePromptTemplate` | 3120 | `get_mcp_analysis` 前置 |
| `get_mcp_analysis` | 💬/✅ | bespoke | mcp-api `/getMCPAnalysis` → analyze-api LLM | 3120→3230 | 必填 `period`；空结果自动重试 1 次；慢 60–180s |
| `get_capital_flow` | ✅ | bespoke | mcp-api `/getCapitalFlow` → GeeGooData | 3120→3300 | DAY 空自动试 WEEK + 重试；仍无数据 skip |
| `get_capital_distribution` | ✅ | bespoke | mcp-api `/getCapitalDistribution` → GeeGooData | 3120→3300 | 空结果自动重试 1 次 |
| `get_bot_yesterday_attitude` | 💬 | bespoke | mcp-api `/getBotYesterdayAttitude` | 3120 | 必填 `bot_id` |
| `get_stock_daily_reports` | 💬 | bespoke | mcp-api `/getStockDailyReports` | 3120 | 建议传 `report_date` |
| `list_today_reports` | ✅ | bespoke | 同上（幂等检查别名） | 3120 | |
| `get_index_signals` | ✅ | HTTP | catalog-api `/getIndexSignalForSkill` | 3210 | DCA 单指标 |
| `get_signal_combinations` | ✅ | HTTP | catalog-api `/getSignalCombinationForSkill` | 3210 | DCA 组合信号 |
| `generate_grid_strategy` | 💬/✅ | HTTP | analyze-api `/generateGridStrategy` | 3230 | 空 payload 重试 1 次；可 fallback 3120 |
| `generate_dca_strategy` | 💬/✅ | HTTP | analyze-api `/generateDCAStrategy` | 3230 | 同上；需先选 `signal_id` |
| `loopback_strategy` | 💬/✅ | HTTP | signal-api `/loopBackStrategy` | 3200 | 需先 `generate_*` 拿参数 |
| `get_bot_log_by_type` | ✅ | HTTP | mcp-api `/getBotLogByType` | 3120 | 必填 `type` + `bot_id` |

### Prompt 模板 CRUD（6）

| Tool | 状态 | 路径 | 端口 |
|------|------|------|------|
| `create/edit/delete_competitor_prompt_template` | ✅ | mcp-api 对应路径 | 3120 |
| `create/edit/delete_etf_prompt_template` | ✅ | mcp-api 对应路径 | 3120 |

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

| Skill | 状态 | 说明 |
|-------|------|------|
| `pre_market` | ✅ | `skills/pre_market/` 有步骤；`geegoo run pre_market` |
| `intraday` | ✅ | 信号触发；`geegoo run intraday --code … --bot-id …` |
| `post_market` | ✅ | 交易日 cron；`geegoo run post_market` |

---

## 八、韧性策略速查（`internal/tools/resilience.go`）

| Tool | 策略 |
|------|------|
| `get_position` / `get_ticker` / `get_broker` | HTTP 空 payload → 等 2s 重试 1 次 |
| `generate_grid_strategy` / `generate_dca_strategy` | 同上 |
| `get_capital_flow` | DAY 空试 WEEK；整轮重试 1 次 |
| `get_capital_distribution` | 空分布重试 1 次 |
| `get_mcp_analysis` | 空 `analysis_result` 重试 1 次 |
| `fetch_stock_news` | finance-news 弱结果 → `web_search` 补充；仍无 → `StatusError` |
| `recall_yesterday_summary` | 本地无文件 → MCP 报告向前查 5 天 |

---

## 九、常踩坑速查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| 资金类 skip | Bot→CN 节点防火墙 / Token / 标的无成交 | `verify_e2e_capital.py`；查 Bot `.env` 路由 |
| 富途三类 skip | OpenD 未配或非交易时段 | 用 `get_current_price` |
| 新闻 `StatusError` | 双源均无标题 | 检查网络；A 股可配 `EASTMONEY_NEWS_APIKEY` |
| generate 503 | analyze-api 或 LLM 未配 | `curl :3230/health` |
| Bot 创建不跑 | GeeGooBot 无 scheduler | 架构缺口，非 Agent bug |

---

## 十、维护

新增 Tool 后同步更新：**本文件**、`tools-tree.md`、`tool-catalog.md`、`catalog/catalog.go`。

核对命令：

```bash
go test ./internal/tools/... -count=1
```
