# 实现状态（2026-07）

> **单一事实来源**：本文与代码不一致时，以 `internal/` + [layers/L2-tools/tools-tree.md](./layers/L2-tools/tools-tree.md) 为准。  
> 业务能力分期见 [phases/README.md](./phases/README.md)。

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 已实现且生产可用 |
| 💬 | 接口正常，需引导用户选择参数后再调（非故障） |
| ⚠️ | 已注册/已接线，但能力降级、依赖外部环境或仅为占位 |
| 📋 | 已登记名称，无业务步骤或目录 |
| ❌ | 未实现 |

---

## 平台内核（Hermes P1–P8）

| 能力 | 状态 | Go 代码 |
|------|------|---------|
| 单二进制 `geegoo` CLI | ✅ | `cmd/geegoo/` |
| HTTP Runtime `:3400` | ✅ | `cmd/agent-runtime/`, `internal/runtimeapi/` |
| ReAct Agent 循环 | ✅ | `internal/agent/`, `internal/runtime/react.go` |
| 上下文压缩（双阈值） | ✅ | `internal/prompt/compressor.go` |
| Tool Registry（82） | ✅ | `internal/tools/` |
| SQLite 会话 + FTS5 | ✅ | `internal/chatsession/sqlite.go`, `internal/infra/db.go` |
| Evidence 可审计报告 | ✅ | `internal/memory/evidence.go` |
| Workflow + Checkpoint | ✅ | `internal/workflow/runner.go` |
| Supervisor 质检 | ✅ | `internal/workflow/supervisor.go` |
| 内置 Scheduler（cron） | ✅ | `internal/scheduler/`, `geegoo scheduler run` |
| Chat TUI（Hermes 风格） | ✅ | `internal/cli/chatui/` |
| Cutover `geegoo verify` | ✅ | `internal/verify/`, `cmd/geegoo/verify.go` |
| Cost Manager | ❌ | 无 `internal/llm/cost` |
| 子 Agent 委派 | ❌ | 无 `spawn_subagent`；见 [subagents.md](./layers/L5-application/subagents.md) |
| Webhook 触发 | ❌ | 无 HTTP webhook 入口 |

---

## 业务能力（Product Phase）

| Phase | 主题 | 状态 | 说明 |
|-------|------|------|------|
| 0 | 平台内核 | ✅ | 见上表 |
| 1 | 盘前 `pre_market` | ✅ | `workflow/premarket.go` + `skills/pre_market/` |
| 2 | 盘后 `post_market` | 📋 | `loader.go` 注册名，**无步骤**、无 `skills/post_market/` |
| 3 | 盘中 `intraday` | 📋 | 同上；富途三接口已接通 ✅ |
| 4 | 按需分析（chat） | ⚠️ | `market` toolset + ReAct；无独立 Skill 包 |
| 5 | 策略 | ✅ | Grid/DCA/MCP 分析 LLM；loopback 原生 Go 回测 |
| 6 | Bot / Reminder 管理 | ⚠️ | CRUD ✅ + schema/prompt；GeeGooBot 无 scheduler |
| 7 | Prompt 模板高级 CRUD | ⚠️ | Tool 已注册；依赖 catalog-api |

---

## Skill

| Skill | 注册 | 步骤 | 资源目录 | 状态 |
|-------|------|------|----------|------|
| `pre_market` | ✅ | PhaseA + PerStock | `skills/pre_market/` | ✅ |
| `intraday` | ✅ | `emptySteps()` | 无 | 📋 |
| `post_market` | ✅ | `emptySteps()` | 无 | 📋 |

---

## Tool 运行态摘要

完整树 → [layers/L2-tools/tools-tree.md](./layers/L2-tools/tools-tree.md)。

| 类别 | 已注册 | 端到端可用 | 主要 ⚠️ |
|------|--------|------------|---------|
| Perception | 10 | 大部分 | 无 Python 时新闻极少 skip |
| Analysis | 22 | 大部分 | `get_mcp_analysis` 质量取决于后端 |
| Decision | 3 | 3 | — |
| Action | 42 | 大部分 | `generate_*` 依赖 analyze-api 部署 |
| Meta | 1 | 1 | — |

### 需引导用户选择（💬）

| Tool | 现象 | Agent 应做 |
|------|------|------------|
| `generate_dca_strategy` | 无 `signal_id` → 101 | 先问单指标/组合 → 列 brief → 用户选 `signal_id` |
| `loopback_strategy` | grid 缺 `grid_param`；dca 缺 `signal` | 先 `generate_*`；567c95c3 已补 schema+prompt |
| `create_*_bot` | 复杂 payload | 先 `generate_*`；向用户确认后 create；写操作需确认 |
| `get_mcp_analysis` | 缺 `period` → 400 | 先 `get_single_prompt_template`，确认 `period` |
| `get_bot_yesterday_attitude` | 缺 `bot_id` | 先 list 对应 Bot，让用户指定 |
| `get_stock_daily_reports` | 缺 `report_date` | 向用户确认日期 |
| `create_pre_market_report` 等 | 缺 `stock_name` 等 | report_workflow 🔒；逐项向用户确认 |

### 已知降级（⚠️）

| Tool | 现象 | 根因 |
|------|------|------|
| `fetch_market_news` / `fetch_stock_news` | 极少 skip | Go RSS/东财回退已实现 |
| `get_ticker` / `get_broker` / `get_position` | — | **已接通** futu_bridge（2026-07-15） |
| `get_mcp_analysis` | — | 经 mcp-api→analyze-api LLM |
| `generate_grid_strategy` / `generate_dca_strategy` | LLM 较慢（60–180s） | analyze-api + promptServer |
| `get_capital_*` | HK/US 常 skip | A 股→GeeGooData CN 节点；港/美→47.80 节点 |
| Bot scheduler | 创建后不自动跑 | GeeGooBot 架构缺口，非 Agent bug |

### 已从 Tool Registry 移除（待 Notify Gateway）

| 原 Tool | 处置 | 说明 |
|---------|------|------|
| `send_feishu_summary` | **待办，非 L2 Tool** | 当前实现为 Agent 本地直连 webhook，未走 GeeGooBot；`pre_market` manifest 未纳入；Notify Gateway（GeeGooBot `internal/notify`）就绪后以 `send_notification` 或 workflow 固定步骤恢复 |

**Notify Gateway 待办（与 scheduler 并列，整体切换前实现）：**

- 落点：GeeGooBot `internal/notify` + HTTP/MCP 出口
- 路由：按 `user.notice`（webhook / FCM / Jpush），非 Agent 全局 `feishu_webhook_url`
- 模板：对齐 TradingBot `BotNotice`（notice_type 0–5）
- Agent 侧：薄转发或 Skill workflow 后置步骤；Supervisor 硬失败告警走 L0，不经 LLM 自由选 tool

---

## 记忆层

| 模块 | 状态 | 说明 |
|------|------|------|
| Session（SQLite + FTS） | ✅ | `recall` Tool 跨会话检索 |
| WorkingMemory | ✅ | Workflow 进度；Executor 内部更新 |
| Evidence | ✅ | 报告 evidence_refs |
| Compaction | ✅ | Hermes 四阶段 |
| Episodic（跨日摘要） | ✅ | `recall_yesterday_summary` 读本地 reports + MCP 回退 |
| Semantic（向量） | ❌ | 刻意不做；见 L3 README |

---

## 基础设施 L0

| 模块 | 状态 | Go |
|------|------|-----|
| EventBus | ✅ | `internal/infra/events.go` |
| StateStore / SQLite | ✅ | `internal/infra/state.go`, `db.go` |
| Checkpoint | ✅ | workflow + `checkpoints` 表 |
| Scheduler | ✅ | `internal/scheduler/`（**非**仅 systemd） |
| Sandbox / WorkspaceGuard | ✅ | `internal/infra/state.go` |
| Secrets / Config | ✅ | `internal/config/` |
| Logging / Tracing | ⚠️ | 文件日志有；分布式 tracing 轻量 |
| Timer 抽象 | ❌ | 规划接口，未单独实现 |

---

## 外部依赖

| 依赖 | MVP 需要？ | 说明 |
|------|------------|------|
| PostgreSQL / Redis | 否 | Agent 状态在 SQLite |
| 向量库 / Embedding | 否 | Semantic 未做 |
| GeeGooBot :3120 | 是 | 报告、Bot、资金等 |
| GeeGooSignal :3200/3210 | 部分 | 搜码、回测、指标 |
| Analyze :3230 | 部分 | 策略/分析（可经 MCP 转发） |
| 富途交易 | 否（盘中才需要） | `get_position` 等 |

---

## 文档维护

状态变更时同步更新：**本文件** → `phases/README.md` → `architecture/README.md` → 相关层 README。
