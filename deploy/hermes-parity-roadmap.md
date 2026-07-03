# GeeGooAgent → Hermes Parity Roadmap

参考 [Hermes Agent 架构](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture) 对标的 Go 化改造路线图。**参考 Hermes 的目录、模块、特性，不照搬其代码**。Provider/Gateway 不追求 Hermes 的 18+ 家，只保留 DeepSeek/OpenAI/Minimax 等少数兼容 OpenAI Chat Completions 的后端。

`deploy/hermes-migration-checklist.md` 保留为 cutover 当天 runbook；本文是优化路线图。

## 一、结构对照

| Hermes 子系统 | Hermes 位置 | GeeGooAgent 现状 | 差距 |
|---|---|---|---|
| Agent 循环 | `run_agent.py` AIAgent | `internal/runtime/react.go` ReActLoop | 有，但只服务 chat；pre_market 走另一条 Workflow.Run |
| Prompt 系统 | `agent/prompt_builder.py` + 压缩 + 缓存 | `internal/chatprompt/prompt.go` 单函数硬编码 | 缺分层组装、缺压缩、缺缓存；且每轮重写 system 破坏缓存 |
| Provider 解析 | `hermes_cli/runtime_provider.py` 18+ | `internal/llm/presets.go` 3 个 | 不多加；但缺 `(provider,model)→(mode,key,base_url)` resolver 抽象 |
| 工具系统 | `tools/registry.py` 70+/28 toolset，导入自注册 | `internal/tools/registry.go` + bespoke + catalog | 无 toolset 分组、无自注册、无 approval/危险检测、无 input/output schema |
| 会话持久化 | `hermes_state.py` SQLite + FTS5 + 血缘 | `internal/chatsession/store.go` 文件 JSON + index manifest | 最大差距之一 |
| Gateway | `gateway/` 20 IM 平台 | 无 | 不需要 IM；agent-runtime HTTP 已替代部分 |
| Cron | `cron/jobs.py` agent 一等公民 | `deploy/systemd/*.timer` shell 级 | 缺 Go 内调度、缺 skill 注入、缺失败重跑 |
| 插件 | `plugins/memory/` `plugins/context_engine/` | 无 | 单租户，YAGNI，后置 |
| ACP | `acp_adapter/` | 无 | 不需要 |
| Skills | `skills/` `optional-skills/` | `skills/pre_market/` + 2 bundled | 有雏形，缺 Go 加载器，steps 硬编码 |
| CLI | `cli.py` HermesCLI | `internal/cli/chatrepl/` + `chatui/` | UI 已对齐 Hermes 风格 |
| 轨迹 | `agent/trajectory.py` ShareGPT | 无 | 可选，后置 |

## 二、Hermes 设计原则 vs GeeGooAgent

| 原则 | 现状 | 建议 |
|---|---|---|
| Prompt 稳定性 | ❌ `SyncChatSystemPrompt` 每轮改 system，破坏 DeepSeek 前缀缓存 | 拆 stable system + 动态 user context |
| 可观测执行 | ✅ `EmitProgress` + UI | 已对齐 |
| 可中断 | ❌ 无 ctx cancel | `context.Context` 贯穿 agent loop 与 tool |
| 平台无关核心 | ⚠️ chat 与 workflow 两条路 | 统一 `Agent.Run(ctx, sess, input)` |
| 松耦合 | ⚠️ MCP/Search/Working 硬编进 Deps | 注册表 + check_fn 门控 |
| Profile 隔离 | ❌ 单 profile | 后置，YAGNI |

## 三、目标 Go 目录

```text
GeeGooAgent/
├── cmd/
│   ├── geegoo/              # CLI: chat / run / resume / setup / doctor / migrate
│   └── agent-runtime/       # HTTP runtime server
├── internal/
│   ├── agent/               # ★ 平台无关核心循环
│   │   ├── agent.go         # Agent = PromptBuilder + Provider + ToolDispatcher + SessionStore
│   │   ├── loop.go          # Run(ctx, sess, input) → 现 ReActLoop 升级
│   │   ├── interrupt.go     # context cancel + signal
│   │   └── trajectory.go    # 可选 ShareGPT 导出
│   ├── prompt/              # ★ 分层 prompt 组装
│   │   ├── builder.go       # SystemBuilder: soul + memory + skills + tools + model_hints
│   │   ├── soul.go          # 静态人格（拆自 chatprompt.go）
│   │   ├── context.go       # 动态上下文作为 user-side context，不改 system
│   │   ├── compressor.go    # ★ 后置：超阈值摘要中间轮次
│   │   └── cache.go         # ★ 后置：前缀缓存断点
│   ├── provider/            # ★ 重命名自 llm/
│   │   ├── provider.go      # Provider interface
│   │   ├── gateway.go       # 重试
│   │   ├── openai.go        # OpenAI 兼容
│   │   ├── presets.go       # DeepSeek/OpenAI/Minimax
│   │   ├── resolver.go      # ★ (provider,model)→(mode,key,base_url)，mode 固定 chat_completions
│   │   └── mock.go
│   ├── tools/
│   │   ├── registry.go
│   │   ├── contract.go      # ★ input/output schema + envelope 校验
│   │   ├── approval.go      # ★ 危险操作检测（写报告/删 bot 前确认）
│   │   ├── toolsets/        # ★ 按域分组
│   │   │   ├── market.go
│   │   │   ├── botmgr.go
│   │   │   ├── report.go
│   │   │   └── meta.go
│   │   ├── catalog/
│   │   └── bespoke.go
│   ├── session/             # ★ 重命名自 chatsession/，迁 SQLite
│   │   ├── store.go         # SessionStore 接口
│   │   ├── sqlite.go        # SQLite + WAL + FTS5
│   │   ├── file.go          # 旧文件实现（迁移期保留）
│   │   ├── migrate.go       # file → sqlite 一次性迁移
│   │   ├── recall.go
│   │   └── schema.sql
│   ├── memory/
│   │   ├── working.go
│   │   ├── models.go
│   │   └── evidence.go      # ★ EvidenceStore 独立落 SQLite
│   ├── workflow/
│   │   ├── runner.go
│   │   ├── premarket.go
│   │   └── supervisor.go    # ★ 跑后质检 + verdict
│   ├── skills/              # ★ Go 加载器
│   │   ├── loader.go        # 扫 skills/*/manifest.yaml
│   │   ├── manifest.go
│   │   └── registry.go
│   ├── scheduler/           # ★ Go 内 cron（替代 systemd timer）
│   │   ├── scheduler.go
│   │   └── jobs.go
│   ├── cli/
│   │   ├── chatrepl/
│   │   ├── chatui/
│   │   └── commands/        # ★ 斜杠命令集中定义
│   ├── infra/
│   │   ├── db.go            # ★ SQLite 句柄 + migration
│   │   ├── state.go         # 保留（迁移期）
│   │   ├── events.go
│   │   └── guard.go
│   ├── clients/mcp/
│   ├── search/
│   ├── config/
│   ├── doctor/
│   ├── auth/
│   └── httpserver/
├── skills/                  # skill 资源
└── deploy/
    ├── systemd/             # 过渡期保留
    └── hermes-parity-roadmap.md
```

## 四、不照搬的部分

| Hermes 模块 | 不做的原因 |
|---|---|
| `gateway/` 20 IM 平台 | 只用 HTTP runtime + CLI |
| `acp_adapter/` IDE 集成 | 不需要 |
| `plugins/memory/` 插件市场 | 单租户 YAGNI |
| `batch_runner.py` 轨迹训练 | 不做训练 |
| `environments/` 6 终端后端 | 工具是 MCP HTTP，不是 shell |
| 18+ Provider | 只保留 DeepSeek/OpenAI/Minimax，统一 OpenAI 兼容 |

## 五、执行阶段

每个 phase 一个 PR，每个 PR 跑 `go test ./...` + 服务器 rebuild。

### P1 — SQLite 地基（最高优先级）

**目标：** 把 Session / Working / Checkpoint / Evidence 从文件 JSON 迁到 SQLite，老调用方零改动。

**改动：**
- `internal/infra/db.go`：SQLite 句柄（`modernc.org/sqlite`，纯 Go 免 CGO），WAL 模式，schema migration
- `internal/session/`（重命名自 `chatsession`）：
  - `store.go` 抽 `SessionStore` 接口
  - `sqlite.go` 实现：表 `chat_sessions`、`session_events`
  - `file.go` 旧文件实现（迁移期）
  - `migrate.go` `geegoo migrate` 命令
  - `schema.sql`
- `internal/memory/evidence.go`：`EvidenceStore`，表 `evidence_records(id PK, run_id, session_id, tool, source, payload_hash, summary, observed_at, payload_json)`
- `internal/infra/state.go`：`StateStore` 接口化，`WorkingStore` / `CheckpointManager` 走 DB
- `cmd/geegoo/migrate.go`：迁移子命令
- `go test ./...` 不回归

**验收：**
- 迁移后 row 数 = 旧文件数
- `geegoo chat` 创建/退出/`recall` 行为不变
- WAL 文件出现，并发写不丢
- `geegoo migrate --dry-run` 能预览

### P2 — 核心统一 + Prompt 稳定性

**目标：** 统一 chat 与 workflow 入口；修 Prompt 缓存失效。

**改动：**
- `internal/agent/`：`Agent.Run(ctx, Session, Input) TurnResult` 作为唯一入口；现 `ReActLoop` 升级为 `agent.loop`
- `internal/prompt/builder.go`：`SystemBuilder` 分层组装；`System()` 在对话中不变
- `internal/prompt/context.go`：Tool 活动摘要作为 **user-side context message**，不再改 system
- `context.Context` 贯穿 `Agent.Run` → `Gateway.Chat` → `Provider.Chat` → `Tool.Handle`
- `workflow/runner.go` 改为：加载 skill → 构造 prompt → 调 `Agent.Run` → 收集 evidence → supervisor
- `internal/provider/`（重命名自 `llm`）：加 `resolver.go`

**验收：**
- `geegoo chat` 与 `geegoo run pre_market` 共用同一个 `Agent.Run`
- DeepSeek 连续两轮调用，system prompt 字节级不变
- Ctrl+C 能中断进行中的 tool 调用

### P3 — 质检 + 真幂等

**目标：** Supervisor 跑后验收；Resume 按 stepKey 幂等；错误分类。

**改动：**
- `internal/workflow/supervisor.go`：`Engine` + `checks.go` + `result.go`（verdict: pass/recoverable/terminal）
- 加载 `skills/pre_market/supervisor_checks.yaml`（或 Go 常量）
- `workflow/runner.go` `RunFrom` 改为按 `StepsCompleted` 集合跳过，不按编号
- 错误分类 `RecoverableError` / `TerminalError`；recoverable 自动补跑
- `cmd/geegoo/resume.go` 加 `--force-step`
- `execution_events` 表补 `retry_count`、`supervisor_verdict`、`started_at`、`ended_at`、`duration_ms`

**验收：**
- 故意漏一步 weekly_analysis → supervisor 标 recoverable → 自动补跑
- `create_pre_market_report` 业务码错 → terminal → 停手告警
- 非交易日 verdict=pass

### P4 — Skill 化

**目标：** `geegoo run <skill>` 通用，不再硬编码 `pre_market`。

**改动：**
- `internal/skills/loader.go`：扫 `skills/*/manifest.yaml`
- `internal/skills/manifest.go`：`workflow` / `template` / `supervisor_checks`
- `skills/pre_market/manifest.yaml` 声明 phase A/B steps（或引用 Go 函数）
- `cmd/geegoo/run.go` 改为 `geegoo run <skill>`，从 SkillRegistry 取
- `skills/intraday/`、`skills/post_market/` 占位

**验收：**
- `geegoo run pre_market` 行为不变
- 新增 skill 只放 manifest + template，不改 Go 代码
- `geegoo skills list` 列出可用 skill

### P5 — Report Synthesis LLM 综合层

**目标：** LLM 只综合，不当数据源。

**改动：**
- `internal/report/synthesis.go`：输入 `StockWorkspace + []EvidenceRef + rules`，调 Gateway 生成 `reason/suggestion/summary`
- Prompt 强制引用 evidence ID，禁止编造；输出 JSON
- `result/confidence` 仍由规则定，LLM 不准改
- 失败回退规则版

**验收：**
- `reason` ≥80 字含具体参数引用
- 报告 evidence_refs 全部能在 SQLite 查到 payload
- LLM 调用失败不阻塞 workflow

### P6 — Tool 契约

**目标：** schema 校验 + toolset 分组 + fixture replay。

**改动：**
- `internal/tools/contract.go`：`OutputSchema` + envelope 校验，`Result.Meta`
- `internal/tools/toolsets/`：按域分组
- `internal/tools/approval.go`：危险操作检测
- 关键工具 fixture replay 测试

**验收：**
- `list_smart_trades` 空 list 但 code=100 标 Skip
- `create_pre_market_report` 写前 approval
- fixture 测试覆盖 5+ 关键工具

### P7 — Scheduler

**目标：** Go 内 cron，与 systemd 并存过渡。

**改动：**
- `internal/scheduler/`：`robfig/cron` 轻量库
- `jobs.json`：`{skill, cron, prompt, platform}`
- 失败时 supervisor 触发重试 job

**验收：**
- `geegoo scheduler start` 跑通盘前
- systemd timer 可禁用
- 失败 30 分钟后自动重跑

### P8 — 验收

**目标：** Parallel verification checklist 升级为可执行。

**改动：**
- `scripts/parallel_verify.py`：拉两边 `createPreMarketReport`，diff 字段
- 抽样 3 只 HK/US/A 股
- 字段完整性矩阵
- `hermes-migration-checklist.md` 每条加"如何验证"命令

**验收：**
- 3 只样本 diff < 阈值
- `bot_id/bot_name/bot_type` 非空率 100%
- evidence_refs 数量 ≥ 规则

## 六、依赖

```text
P1 (SQLite) ──┬──> P3 (Supervisor/Resume)
              ├──> P5 (Synthesis，需查 evidence)
              └──> P2 (核心统一，session DB 落地)
P2 ──> P4 (Skill 化，agent.Run 通用)
P3 ──> P7 (Scheduler，supervisor 触发重跑)
P6 独立，可与 P2-P5 并行
P8 依赖 P1-P6
```

## 七、Provider 取舍

只保留：
- `deepseek`（默认，支持 thinking）
- `openai`（兼容）
- `minimax`

统一 `api_mode = chat_completions`，`resolver.go` 保留 `(provider, model) → (mode, key, base_url)` 签名以备扩展，但 mode 当前固定。不实现 Hermes 的 codex_responses / anthropic_messages。

## 八、技术选型

| 项 | 选型 | 理由 |
|---|---|---|
| SQLite 驱动 | `modernc.org/sqlite` | 纯 Go，免 CGO，服务器 cross-compile 友好 |
| FTS5 | SQLite 内置扩展 | modernc 编译时含 |
| Cron | `robfig/cron/v3` | 轻量，标准库风格 |
| YAML | `gopkg.in/yaml.v3` | skill manifest |
| 不引入 | jsonschema 库 | tool schema 用 Go struct + 手写校验 |
