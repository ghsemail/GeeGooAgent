# Toolset 与扩展

> Tool 注册机制见 [registry.md](./registry.md)。运行态可用性见 [tools-status.md](./tools-status.md)。

## Toolset（Hermes 风格）

定义：`internal/tools/toolset.go` + `domains.go`

| Toolset ID | 中文 | 默认 chat | 工具数 |
|------------|------|-----------|--------|
| `market` | 行情与分析 | ✅ | 18 |
| `strategy` | 策略生成与回测 | ✅ | 3 |
| `bot_manager` | 交易 Bot | ✅ | 20 |
| `reminder_manager` | 提醒 Bot | ✅ | 15 |
| `report_query` | 报告查询 | ✅ | 13 |
| `report_workflow` | 报告 Workflow | ❌ | 8 |
| `prompt_template` | Prompt 模板 CRUD | ❌ | 6 |

**默认 chat 白名单：69**（5 个 ChatDefault toolset，减去 7 个 workflow 独占 tool）。

Chat 切换：`/toolsets market,strategy` · `/toolsets default` · `/toolsets prompt_template`（高级）

**workflow 独占（7，默认不进 chat）**：`get_report_bot_codes`、`create_pre_market_report`、`save_local_report`、`write_execution_log`、`read_working_state`、`recall_yesterday_summary`、`list_today_post_market_reports`。

**workflow 共享（1）**：`get_bot_yesterday_attitude`（同时在 `market`，默认 chat 可用）。

Workflow（`geegoo run`）不按 toolset 过滤，步骤在 `workflow/premarket.go` 硬编码。

## 五类 Taxonomy

| 类 | 代表 |
|----|------|
| Perception | `search_code`, `get_current_price`, `web_search`, `fetch_*_news` |
| Analysis | `get_mcp_analysis`, `get_capital_flow`, `get_index_signals` |
| Decision | `recall`, `read_working_state` |
| Action | `create_*_report`, `create_*_bot`, `save_local_report` |
| Meta | `write_execution_log` |

**无 Bash Tool。**

## 关键机制

| 机制 | 文件 |
|------|------|
| ApprovalGate | `approval.go` — chat 写操作需确认（含 `edit_*` Prompt 模板） |
| ClassifyHTTPPayload | `contract.go` — 空 data → skipped |
| NeedsMCPToken | `catalog/token.go` |
| HTTPBackends | `httpbackend.go` — 按 tool 选 :3120/:3200/:3210 |

## HTTP 路由摘要

| 端口 | Tools |
|------|-------|
| 3120 MCP | 报告、Bot、资金、`get_mcp_analysis`（bespoke，Bot 内转发 analyze-api）、`fetch_*_news`（→ GeeGooData）、策略 fallback |
| 3200 Signal | `search_code`, `loopback_strategy` |
| 3210 Catalog | `get_index_signals`, `get_signal_combinations` |
| 3230 Analyze | `generate_grid_strategy`, `generate_dca_strategy`（Agent 直连，失败可 fallback 3120） |
| 3300 Data | 现价、资金、新闻（经 Bot/bespoke，Agent 不直连） |

## Skill Pack → Tool 子集

| Skill Pack | Tool 组 | Phase |
|------------|---------|-------|
| `pre_market` | 感知+分析+报告 workflow | 1 |
| `post_market` | pre 子集 + post 报告 | 2 |
| `intraday` | 持仓 + intraday 报告 | 3 |
| `on_demand_analysis` | market 核心 | 4 |
| `strategy` | §2.3 策略 | 5 |
| `bot_manager` | Bot CRUD + logs | 6 |

manifest 白名单：`skills/<skill>/manifest.yaml`（`pre_market` / `intraday` / `post_market`）。Skill 文档 → [L5 skills](../L5-application/skills.md)。

## 扩展：新增 Tool

1. `catalog/catalog.go` 增加 `HTTPSpec`，或 `bespoke.go` 注册
2. 更新 `domains.go` / `toolset.go`
3. GeeGooBot 注册 MCP 路由（若新 HTTP）
4. 同步 [tools-status.md](./tools-status.md) + [tool-catalog.md](./tool-catalog.md)

## 与 Skill 的关系

| | Tool | Skill |
|---|------|-------|
| 粒度 | 单次调用 | 多步工作流 |
| Chat | toolset 白名单 | 不直接暴露 |
| Run | workflow 硬编码步骤 | `geegoo run <skill>` |

领域映射：[domains/geegoo-skill-mapping.md](../../domains/geegoo-skill-mapping.md)
