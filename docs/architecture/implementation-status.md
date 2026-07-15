# 实现状态（2026-07）

> **单一事实来源**：本文与代码不一致时，以 `internal/` + [layers/L2-tools/tools-tree.md](./layers/L2-tools/tools-tree.md) 为准。  
> 业务能力分期见 [phases/README.md](./phases/README.md)。

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 已实现且生产可用 |
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
| 3 | 盘中 `intraday` | 📋 | 同上；富途三接口见 Tool ⚠️ |
| 4 | 按需分析（chat） | ⚠️ | `market` toolset + ReAct；无独立 Skill 包 |
| 5 | 策略 | ⚠️ | Tool 已注册；Analyze/Signal 为简化实现 |
| 6 | Bot / Reminder 管理 | ⚠️ | CRUD Tool ✅；GeeGooBot 侧无 Bot scheduler |
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
| Perception | 10 | 大部分 | 富途三接口 Noop；无 Python 时新闻 skip |
| Analysis | 22 | 大部分 | `get_mcp_analysis` 质量取决于后端 |
| Decision | 3 | 3 | — |
| Action | 42 | 大部分 | `generate_*` 依赖 analyze-api 部署 |
| Meta | 1 | 1 | — |

### 已知降级 Tool

| Tool | 现象 | 根因 |
|------|------|------|
| `fetch_market_news` / `fetch_stock_news` | 无 Python/脚本时 skip | 需 `skills/bundled/finance-news` + Python |
| `recall_yesterday_summary` | 无昨日报告时 skip | 正常；读 `reports/{date}/{code}-premarket.md` |
| `get_ticker` / `get_broker` / `get_position` | 空/Skip | MCP 富途未配置 |
| `get_mcp_analysis` | 输出质量因后端而异 | 优先 analyze-api :3230 |
| `generate_grid_strategy` / `generate_dca_strategy` | 502/404 | analyze-api 未部署时回退 Bot:3120 |
| `loopback_strategy` | 确定性简化回测 | Signal/Go 非完整回测引擎 |
| `send_feishu_summary` | 未配 webhook 时 skip | 配置 `feishu_webhook_url` 或 `GEEGOO_FEISHU_WEBHOOK_URL` |

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
