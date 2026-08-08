# Toolset 与扩展

> Tool 注册机制见 [registry.md](./registry.md)。运行态可用性见 [tools-status.md](./tools-status.md)。

## Toolset（Hermes 风格）

定义：`internal/tools/toolset.go` + `domains.go`

| Toolset ID | 中文 | 默认 chat | 工具数 |
|------------|------|-----------|--------|
| `market_data` | 行情与账户 | ✅ | 5 |
| `research` | 研究与分析 | ✅ | 4 |
| `info_search` | 信息检索 | ✅ | 4 |
| `agent_meta` | Agent 元能力 | ✅ | 8 |
| `strategy` | 策略与信号 | ✅ | 5 |
| `trading_bot` | 交易机器人 | ✅ | 15 |
| `hedge_bot` | 对冲机器人 | ✅ | 5 |
| `reminder_manager` | 提醒机器人 | ✅ | 15 |
| `report_query` | 报告查询 | ✅ | 15 |
| `report_workflow` | 报告 Workflow | ❌ | 8 |
| `prompt_template` | Prompt 模板 CRUD | ❌ | 6 |

**兼容别名**：`market` → `market_data` + `research` + `info_search`；`bot_manager` → `trading_bot` + `hedge_bot`。

**默认 chat 白名单：74**（9 个 ChatDefault toolset，减去 7 个 workflow 独占 tool + `recall`）。

Chat 切换：`/toolsets market_data,research` · `/toolsets default` · `/toolsets prompt_template`（高级）

**workflow 独占（7，默认不进 chat）**：`get_report_bot_codes`、`create_stock_premarket_report`、`save_local_report`、`write_execution_log`、`read_working_state`、`recall_yesterday_summary`、`list_today_stock_postmarket_reports`。

**workflow 共享（1）**：`get_bot_yesterday_attitude`（同时在 `report_query`，默认 chat 可用）。

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
| `premarket_market` | 感知+分析+报告 workflow | 1 |
| `postmarket_stock` | pre 子集 + post 报告 | 2 |
| `intraday` | 持仓 + intraday 报告 | 3 |
| `on_demand_analysis` | market 核心 | 4 |
| `strategy` | §2.3 策略 | 5 |
| `bot_manager` | Bot CRUD + logs | 6 |

manifest 白名单：`skills/<skill>/manifest.yaml`（`premarket_market` / `intraday` / `postmarket_stock`）。Skill 文档 → [L5 skills](../L5-application/skills.md)。

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
