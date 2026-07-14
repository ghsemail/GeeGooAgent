# GeeGooAgent 架构

本页是 GeeGooAgent 内部结构的顶层导图。用它在代码库中定位自己，然后深入各子系统专项文档了解实现细节。

> 对照 [Hermes Agent 架构](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture)。GeeGooAgent 参考 Hermes 的目录、模块、特性设计，但不照搬其代码；不实现 IM 平台 / ACP / 插件市场 / 轨迹训练等 GeeGoo 不需要的部分。完整对比见 [`../../deploy/hermes-parity-comparison.md`](../../deploy/hermes-parity-comparison.md)。

## 系统概览

```text
┌─────────────────────────────────────────────────────────────────────┐
│                        Entry Points                                  │
│                                                                      │
│  CLI (cmd/geegoo)    HTTP Runtime (cmd/agent-runtime)   Go Library   │
│  chat / run /        /v1/chat/completions                internal/*  │
│  resume / scheduler  /health /ready                                 │
│  verify / migrate / skills / doctor                                  │
└──────────┬──────────────┬───────────────────────┬───────────────────┘
           │              │                       │
           ▼              ▼                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Agent (internal/agent)                           │
│                                                                     │
│  Agent.Run(ctx, session, input)  ← 平台无关核心                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │ Prompt       │  │ Provider     │  │ Tool         │               │
│  │ (chatprompt +│  │ Resolution   │  │ Dispatch     │               │
│  │  RuntimeMsgs)│  │ (llm/        │  │ (tools/      │               │
│  │              │  │  presets+    │  │  registry+   │               │
│  │ stable system│  │  gateway+    │  │  catalog+    │               │
│  │ + dynamic ctx│  │  openai)     │  │  bespoke)    │               │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘               │
│         │                 │                 │                       │
│  ┌──────┴───────────────────────────────────────────────────────┐   │
│  │ Context Compressor (internal/prompt)                         │   │
│  │ Hermes-style token-threshold compression before LLM rounds   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│         │                 │                 │                       │
│  ┌──────┴───────┐  ┌──────┴───────┐  ┌──────┴───────┐               │
│  │ DeepSeek     │  │ 3 providers  │  │ ~82 tools    │               │
│  │ thinking +   │  │ chat_compl.  │  │ approval gate│               │
│  │ ctx cancel   │  │ retries      │  │ empty-success│               │
│  │ interrupt    │  │              │  │ detection    │               │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘               │
└─────────┴─────────────────┴─────────────────┴───────────────────────┘
           │                                    │
           ▼                                    ▼
┌─────────────────────┐              ┌──────────────────────┐
│ Session + Evidence  │              │ Tool Backends         │
│ (SQLite + WAL+FTS5) │              │ GeeGooBot MCP :3120   │
│ infra/db.go         │              │ Signal :3210/:3230    │
│ chatsession/sqlite  │              │ Data :3300            │
│ memory/evidence     │              │ DuckDuckGo web search │
└─────────────────────┘              └──────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│              Workflow + Skills + Supervisor + Scheduler              │
│  workflow/runner  →  skills/registry  →  supervisor verdict          │
│  report/synthesis (evidence-only LLM)                                │
│  scheduler/ (cron + retry)        verify/ (cutover acceptance)       │
└─────────────────────────────────────────────────────────────────────┘
```

## 目录结构

```text
GeeGooAgent/
├── cmd/
│   ├── geegoo/              # CLI 入口：chat / run / resume / setup / doctor / migrate / skills / scheduler / verify
│   │   ├── main.go          # 子命令分发
│   │   ├── chat.go          # 交互式 chat
│   │   ├── run.go           # geegoo run <skill>
│   │   ├── ops.go           # setup / update / resume
│   │   ├── migrate.go       # file → SQLite 一次性迁移
│   │   ├── skills.go        # geegoo skills list
│   │   ├── scheduler.go     # geegoo scheduler run|list
│   │   └── verify.go        # geegoo verify（cutover 验收）
│   └── agent-runtime/       # HTTP runtime server（/v1/chat/completions, /health）
│
├── internal/
│   ├── agent/               # ★ 平台无关核心
│   │   ├── agent.go         # Agent = Loop + Gateway + Executor + Registry；Run(ctx, sess, input)
│   │   └── agent_test.go
│   │
│   ├── chatprompt/          # 系统 prompt（静态人格 + 路由规则）
│   │   └── prompt.go
│   │
│   ├── prompt/              # 上下文压缩（Hermes-style，LLM 轮前触发）
│   │   ├── compressor.go    # 四阶段压缩 + ShouldCompress
│   │   ├── summary.go       # 辅助 LLM 结构化摘要
│   │   └── estimate.go      # token 估算（chars/4）
│   │
│   ├── llm/                 # Provider 层（roadmap 计划重命名为 provider/）
│   │   ├── types.go         # Message / Provider interface / ToolSchema
│   │   ├── openai.go        # OpenAI 兼容（DeepSeek/OpenAI/Minimax），thinking + reasoning_content
│   │   ├── gateway.go       # 重试网关（ctx 可取消）
│   │   ├── presets.go       # 3 provider 预设 + 模型目录 + thinking 解析
│   │   └── mock.go          # 测试用
│   │
│   ├── tools/               # 工具系统
│   │   ├── registry.go      # Registry + Context（含 Ctx/Interactive/Approved）+ Result（含 Meta）
│   │   ├── bootstrap.go     # HTTP catalog 转发 + ApprovalGate + Meta + 空成功检测
│   │   ├── bespoke.go       # 手写工具（search_code/web_search/analysis/...）
│   │   ├── contract.go      # ClassifyHTTPPayload（code=100 但空 → Skip）+ MetaFromEnvelope
│   │   ├── approval.go      # 危险操作门控（create_/update_/delete_/switch_）
│   │   ├── domains.go       # ChatToolNames 白名单 + 按域分组
│   │   └── catalog/         # HTTP 转发规格 + NeedsMCPToken
│   │
│   ├── chatsession/         # 会话持久化（SessionStore 接口）
│   │   ├── store.go         # ChatSession + SessionStore 接口 + 文件实现
│   │   ├── sqlite.go        # SQLiteSessionStore + FTS5 全文检索
│   │   ├── recall.go        # 跨会话 recall（SearchPastSessions）
│   │   └── prompt_stability_test.go
│   │
│   ├── memory/              # 工作记忆 + 证据
│   │   ├── models.go        # PreMarketWorking + EvidenceRef + PayloadHash
│   │   ├── working.go       # WorkingStore + Apply（工具结果落工作记忆）
│   │   └── evidence.go      # EvidenceStore（SQLite，可审计 payload + hash）
│   │
│   ├── workflow/            # 确定性工作流
│   │   ├── runner.go        # Runner.Run/RunFrom（按 CompletedStepKeys 幂等）+ SynthesizerProvider
│   │   ├── premarket.go     # Phase A/B steps + BuildReportContent + BuildCreateReportArgs
│   │   ├── supervisor.go    # Engine.Verify → verdict pass/recoverable/terminal
│   │   └── errors.go        # StepError（Recoverable/Terminal）+ classifyError
│   │
│   ├── skills/              # Skill 注册表
│   │   ├── registry.go      # Spec + Registry（Register/Get/List）
│   │   └── loader.go        # RegisterBuiltins（pre_market + intraday/post_market 占位）
│   │
│   ├── scheduler/           # Go 内 cron
│   │   ├── scheduler.go     # Runner（robfig/cron）+ supervisor 驱动退避重试
│   │   └── jobs.go          # Job + jobs.json 读写 + DefaultJobs
│   │
│   ├── report/              # 报告综合
│   │   └── synthesis.go     # LLM evidence-only 综合（reason/suggestion/summary；result/confidence 规则锁定）
│   │
│   ├── verify/              # Cutover 验收
│   │   └── verify.go        # VerifyReport（字段完整 + 枚举 + reason 长度 + evidence_refs）+ CompletenessMatrix
│   │
│   ├── infra/               # 底层基础设施
│   │   ├── db.go            # SQLite 句柄（modernc.org/sqlite，纯 Go 免 CGO）+ WAL + schema migration
│   │   ├── schema.sql       # 7 表 + FTS5（chat_sessions/session_events/evidence_records/working_state/checkpoints/execution_events）
│   │   ├── state.go         # 文件 StateStore（迁移期保留）+ CheckpointManager + WorkspaceGuard
│   │   └── events.go        # EventBus（L0 解耦）
│   │
│   ├── clients/mcp/         # GeeGooBot MCP 客户端（Bearer + mcp_token body）
│   ├── search/              # 免费网页搜索（DuckDuckGo HTML）
│   ├── config/              # 配置加载 + 端点解析 + 路径
│   ├── doctor/              # 健康检查
│   ├── auth/                # Bearer 中间件
│   ├── httpserver/          # runtime HTTP mux
│   ├── runtimeapi/          # /v1/chat/completions handler
│   └── cli/
│       ├── chatrepl/        # 交互式 REPL + 斜杠命令 + 信号中断
│       └── chatui/          # Hermes 风格终端 UI（banner/theme/markdown/tools_display）
│
├── skills/                  # Skill 资源（manifest + template + supervisor_checks）
│   ├── pre_market/
│   └── bundled/
│
├── deploy/
│   ├── systemd/             # 过渡期 systemd unit
│   ├── hermes-parity-roadmap.md       # P1–P8 路线图（已完成）
│   ├── hermes-parity-comparison.md    # Hermes 对比
│   └── hermes-migration-checklist.md  # cutover runbook
│
├── rules/                   # 报告格式 / 态度映射 / API 路由规则
└── docs/                    # 本文档
```

## 数据流

### CLI Chat 会话

```text
用户输入 → chatrepl.runTurn()
  → Chat.SyncChatSystemPrompt()（system 保持稳定，不改内容）
  → Chat.RuntimeMessages()（在最后一条 user 前注入动态 Tool 活动 context）
  → Agent.Run(ctx, session, text, toolCtx, schemas)
    → Gateway.Chat(ctx, ...)（重试，ctx 可取消）
    → tool_calls? → Executor.Execute → Registry.Execute → MCP/Search
    → 循环直到无 tool_call
  → 最终响应 → chatui 渲染 → 保存到 SessionStore（SQLite 或文件）
  → Ctrl+C 中断当前回合，下一回合可继续
```

### Pre-market Workflow

```text
geegoo run pre_market
  → App.RunSkill("pre_market")（从 skills registry 查 Spec）
  → Workflow.Run(phaseA, perStock)
    → 每步 processStep：Execute → Working.Apply → write_execution_log → checkpoint
    → 按 CompletedStepKeys 幂等跳过（resume 不因 bot 列表变化而错位）
    → Recoverable 错误自动重试 1 次；Terminal 直接 fail
  → finishWithSupervisor：Engine.Verify → verdict（pass/recoverable/terminal）
  → 报告生成：BuildCreateReportArgs 调 report.Synthesizer（evidence-only LLM）
    → result/confidence 规则锁定；reason/suggestion/summary 由 LLM 综合
    → LLM 失败回退规则版，不阻塞
  → create_pre_market_report 入库 + save_local_report 留档
```

### Scheduler 任务

```text
geegoo scheduler run
  → LoadJobs(jobs.json)（默认 pre_market 工作日 08:00）
  → robfig/cron 注册 enabled jobs
  → tick → runJob → App.RunSkill(skill) → supervisor verdict
  → pass：不动
  → recoverable/terminal：指数退避重试（30m → 60m，最多 2 次）
  → 记录 last_run/last_verdict 到 jobs.json
  → SIGTERM/SIGINT 优雅停机
```

### HTTP Runtime

```text
POST /v1/chat/completions（Bearer + X-MCP-Token）
  → runtimeapi.chatCompletions
  → Agent.Run(r.Context(), session, lastUser, ctx, schemas)
  → 返回 OpenAI 兼容 JSON
```

## 主要子系统

### Agent 循环

`internal/agent/agent.go` 中的 `Agent.Run(ctx, session, input, toolCtx, schemas)` 是平台无关核心，CLI chat、HTTP runtime 共用。封装 ReAct loop + Gateway + Executor + Registry。支持 `context.Context` 取消（Ctrl+C 中断进行中的 LLM/tool 调用）。

### Prompt 系统

`internal/chatprompt/prompt.go` 提供稳定 system prompt（人格 + Tool 路由规则 + 记忆规则）。`ChatSession.RuntimeMessages()` 在最后一条 user message 前注入动态 Tool 活动 context（user 角色），**system message 跨轮字节不变**，保 DeepSeek/OpenAI 前缀缓存命中率。DeepSeek thinking 模式通过 `llm/openai.go` 的 `thinking`/`reasoning_effort`/`reasoning_content` 解析接入。

`internal/prompt/compressor.go` 在回合开始（默认 85% hygiene）与每轮 LLM 调用前（默认 50%）按 token 阈值触发 Hermes-style 四阶段压缩；`context_length` 可按当前模型自动解析。详见 [`layers/L3-memory/compaction.md`](layers/L3-memory/compaction.md)。

### Provider 解析

`internal/llm/presets.go` 提供 3 个 provider（DeepSeek/OpenAI/Minimax），统一 OpenAI 兼容 `chat_completions` mode。`BuildProviderFromLLMFields(provider, model, token, thinking, effort)` 解析为 `(provider, key, base_url, thinking_enabled)`。不实现 Hermes 的 18+ provider / codex_responses / anthropic_messages（按需精简）。

### 工具系统

`internal/tools/registry.go` 中央注册表，**82** 个工具（catalog HTTP 转发 + bespoke 手写）。完整树形图：[geegoo-agent-tools-tree.md](../reference/geegoo-agent-tools-tree.md)。关键能力：
- `ApprovalGate`：`create_/update_/delete_/switch_` 工具在交互式 chat 且未确认时 Skip；workflow 路径不受影响
- `ClassifyHTTPPayload`：API 返回 code=100 但 data 空 → `StatusSkip` + 数据缺口 note（避免「看似成功但数据不可用」）
- `Result.Meta`：每次 HTTP 工具带 `api_code/duration_ms`
- `catalog.NeedsMCPToken`：除 search_code/get_index_signals/get_signal_combinations 外默认注入 mcp_token

### 会话持久化

`internal/chatsession/sqlite.go` + `internal/infra/db.go`：SQLite + WAL + FTS5。表 `chat_sessions`/`session_events`/`evidence_records`/`working_state`/`checkpoints`/`execution_events`。`SessionStore` 接口允许文件实现（迁移期）和 SQLite 实现共存。`geegoo migrate` 一键从文件 JSON 迁到 SQLite。`recall` 跨会话检索历史查价/搜索活动。

### Evidence Store

`internal/memory/evidence.go`：每条工具结果落 `evidence_records`（id/run_id/session_id/tool/source/payload_hash/summary/observed_at/payload_json）。报告只存 evidence IDs；`VerifyPayload` 重新 hash 校验。这是 GeeGoo 相对 Hermes 的差异化能力——报告可审计、可追溯。

### Workflow + Supervisor

`internal/workflow/runner.go` 确定性步骤执行。`RunFrom` 按 `CompletedStepKeys`（命名 step key）幂等跳过，不按扁平编号——bot 列表变化不会导致 step 错位。`supervisor.go` 跑后验收：phase done、本地 md 存在、API report_id/bot 字段、evidence_refs；输出 verdict `pass`/`recoverable`/`terminal`。recoverable 列出 MissingSteps 可补跑；terminal（status=failed）停手告警。非交易日 verdict=pass。

### Report Synthesis

`internal/report/synthesis.go`：LLM 只综合 reason/suggestion/summary，**严禁编造数据**，prompt 强制引用 evidence ID，reason ≥80 字。`result`/`confidence` 锁定规则（attitude→result，evidence 数→confidence），LLM 不能翻转决策。LLM 失败回退规则版，不阻塞 workflow。

### Skills

`internal/skills/registry.go` + `loader.go`：`geegoo run <skill>` 从 registry 查 Spec（PhaseA/PerStock 函数 + template 路径）。pre_market 实步注册；intraday/post_market 占位。新增 skill 只在 RegisterBuiltins 加一项 + 实现步骤函数，无需改 cmd/app。

详见 [`tools-and-skills.md`](./tools-and-skills.md)。

### Scheduler

`internal/scheduler/scheduler.go`：robfig/cron/v3 驱动。jobs.json 存 `{name, skill, cron, enabled, last_run, last_verdict}`。tick 跑 `App.RunSkill`，按 supervisor verdict 决定是否退避重试。`geegoo scheduler run` 长驻，SIGTERM 优雅停。替代外部 systemd timer，能在 agent 内做「盘前失败 → 30 分钟后重跑」。

### Cutover 验收

`internal/verify/verify.go` + `cmd/geegoo/verify.go`：`geegoo verify --codes <list> [--date <D>]` 拉 `getStockDailyReports`，逐条检查 bot_id/bot_name/bot_type 非空、result/confidence/suggestion 枚举合法、reason ≥80 字、evidence_refs 非空，输出字段完整率矩阵，失败 exit 1。

## 设计原则

| 原则 | 实践 |
| --- | --- |
| Prompt 稳定性 | system message 对话中字节不变；动态 context 作为 user-side 注入；除 `/think`/`/model` 外不破坏缓存 |
| 可观测执行 | 每次工具调用通过 `EmitProgress` 对用户可见（chatui spinner + 工具预览） |
| 可中断 | `context.Context` 贯穿 Agent→Gateway→Provider→Tool；Ctrl+C 中断进行中回合，下回合可继续 |
| 平台无关核心 | 单一 `Agent.Run` 同时服务 CLI chat、HTTP runtime、workflow LLM 合成；平台差异在入口点 |
| LLM 不当数据源 | 报告 result/confidence 规则锁定；LLM 只综合 evidence 已有数据；失败回退规则版 |
| 报告可审计 | 每条结论可追溯到 evidence_records 原始 payload + hash |
| 幂等 resume | 按 step key 跳过，不按编号；bot 列表变化不导致错位 |
| 质检驱动 | supervisor verdict 决定 pass/recoverable/terminal；recoverable 自动补跑；scheduler 据此退避重试 |

## 文件依赖链

```text
infra/db.go + schema.sql  （无依赖——SQLite 句柄）
       ↑
chatsession/sqlite.go, memory/evidence.go  （依赖 infra.DB）
       ↑
tools/registry.go  （无依赖——被所有工具导入）
       ↑
tools/bootstrap.go + bespoke.go  （注册时调用 registry.Register）
       ↑
runtime/react.go  （依赖 tools + llm）
       ↑
agent/agent.go  （封装 runtime + llm + tools）
       ↑
app/app.go  （组装所有依赖）
       ↑
cmd/geegoo/* + cmd/agent-runtime/*  （入口）
```

工具注册发生在 `app.LoadFromConfigPath` 调用 `tools.RegisterAll` 时，早于任何 agent 实例创建。

## Tools 与 Skills（速览）

GeeGooAgent 能力由 **Skill（任务包）** 与 **Tool（原子 API）** 两层协作：

| 层 | 数量 | 文档 |
|----|------|------|
| 已注册 Tool | **82** | [`../reference/geegoo-agent-tools-tree.md`](../reference/geegoo-agent-tools-tree.md) |
| 内置 Skill | 3（pre_market 完整） | [`tools-and-skills.md`](./tools-and-skills.md) |
| Chat toolset | 6 组 | `internal/tools/toolset.go` |

- **Chat**：LLM 经 ReAct 调用 toolset 白名单内 Tool  
- **Workflow**：`geegoo run pre_market` 硬编码步骤，不依赖 LLM 编排顺序  
- **常踩坑**：新闻 Tool skipped、富途三接口 Noop、`switch_bot` 未注册

综合导读 → **[tools-and-skills.md](./tools-and-skills.md)**

## 推荐阅读顺序

如果你是第一次接触代码库：

1. **本页** — 整体定位
2. [`README.md`](./README.md) — 文档书籍目录
3. [`entrypoints.md`](./entrypoints.md) — CLI / HTTP / Scheduler
4. [`tools-and-skills.md`](./tools-and-skills.md) — Tool + Skill 体系
5. [`layers/L4-runtime/agent-loop.md`](./layers/L4-runtime/agent-loop.md) — ReAct 循环
6. [`layers/L3-memory/compaction.md`](./layers/L3-memory/compaction.md) — 上下文压缩
7. [`../reference/geegoo-agent-tools-tree.md`](../reference/geegoo-agent-tools-tree.md) — 哪些 Tool 能用
8. [`domains/README.md`](./domains/README.md) — GeeGoo API 映射
9. [`../../deploy/hermes-parity-roadmap.md`](../../deploy/hermes-parity-roadmap.md) — P1–P8 交付记录

深入代码：`internal/agent/agent.go` → `workflow/runner.go` → `infra/schema.sql`
