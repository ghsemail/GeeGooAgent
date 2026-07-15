# 入口点（Entry Points）

GeeGooAgent 与 Hermes 一样，**平台差异在入口点，不在 Agent 核心**。所有路径最终汇聚到 `internal/agent.Agent.Run` 或 `internal/workflow.Runner`。

## 入口一览

| 入口 | 二进制 / 命令 | 用途 | 文档 |
|------|---------------|------|------|
| **CLI Chat** | `geegoo chat` | 交互式对话、按需分析、Bot 管理 | 本文 §CLI |
| **CLI Workflow** | `geegoo run <skill>` | 确定性盘前/盘后工作流 | [workflow-engine.md](./layers/L4-runtime/workflow-engine.md) |
| **CLI Scheduler** | `geegoo scheduler run` | 长驻 cron，按 skill 定时跑 | [scheduler.md](./layers/L0-infrastructure/scheduler.md) |
| **HTTP Runtime** | `agentRuntimeServer` `:3400` | OpenAI 兼容 `/v1/chat/completions` | 本文 §HTTP |
| **运维** | `geegoo doctor` / `setup` / `migrate` / `verify` | 健康检查、配置、迁移、验收 | 本文 §运维 |

```text
                    ┌─────────────────────────────────────┐
                    │         Entry Points                 │
                    ├─────────────┬───────────────┬───────┤
                    │ geegoo chat │ geegoo run    │ :3400 │
                    │ + toolsets  │ pre_market    │ HTTP  │
                    │ + /commands │ scheduler     │       │
                    └──────┬──────┴───────┬───────┴───┬───┘
                           │              │           │
                           ▼              ▼           ▼
                    Agent.Run()    workflow.Runner   Agent.Run()
                    (ReAct)        (确定性步骤)       (ReAct)
```

---

## CLI（`cmd/geegoo`）

### 子命令

| 命令 | 文件 | 说明 |
|------|------|------|
| `chat` | `chat.go` + `cli/chatrepl` | 主交互入口；Bubble Tea UI（`cli/chatui`） |
| `run <skill>` | `run.go` | `App.RunSkill` → workflow |
| `resume` | `ops.go` | 从 checkpoint 续跑 workflow |
| `setup` | `ops.go` | 写 `~/.geegoo/config.json` |
| `doctor` | `ops.go` | 出站 MCP/Signal/Data/Runtime 探活 |
| `migrate` | `migrate.go` | 文件 Session → SQLite 一次性迁移 |
| `skills list` | `skills.go` | 列出 `internal/skills` 注册表 |
| `scheduler run\|list` | `scheduler.go` | cron 守护 |
| `verify` | `verify.go` | 盘前报告字段完整率验收 |

### Chat 数据流

```text
用户输入
  → chatrepl.runTurn()
  → Chat.SyncChatSystemPrompt()     # system 字节稳定
  → Chat.RuntimeMessages()          # 动态 context 作 user 注入
  → Agent.Run(ctx, session, text, toolCtx, schemas)
       → Gateway.Chat (可取消)
       → tool_calls → Registry.Execute → MCP/Search
  → chatui 渲染
  → SessionStore.Save (SQLite)
```

### Chat 斜杠命令（Hermes 风格）

实现：`internal/cli/chatui/commands.go`

| 命令 | 作用 |
|------|------|
| `/tools` | 按域列出当前可用 Tool |
| `/toolsets` | 切换 toolset（market / strategy / bot_manager …） |
| `/model` | 切换 provider + model |
| `/think` | DeepSeek thinking 开关 |
| `/sessions` | 多会话管理（ChatTUI） |
| `/quit` | 退出 |

### Toolset 白名单

默认 chat 加载 5 个 toolset（不含 `report_workflow`）。详见 [layers/L2-tools/toolsets.md](./layers/L2-tools/toolsets.md)。

---

## HTTP Runtime（`cmd/agent-runtime`）

| 路由 | 说明 |
|------|------|
| `GET /health` | 探活 |
| `GET /ready` | 就绪 |
| `POST /v1/chat/completions` | OpenAI 兼容；Bearer + `X-MCP-Token` |

处理链：`internal/runtimeapi/handler.go` → `Agent.Run`。

适用场景：远程调用 Agent、与外部编排系统集成；**不**替代 GeeGooBot `mcp-api`（3120 是 Tool 后端，不是 Agent 入口）。

---

## Scheduler

`geegoo scheduler run` 长驻进程：

1. 读 `jobs.json`（默认工作日 08:00 `pre_market`）
2. `robfig/cron` 触发 `App.RunSkill`
3. 按 `supervisor` verdict 指数退避重试（30m → 60m，最多 2 次）
4. SIGTERM 优雅停机

替代外部 systemd timer 做「失败后自动补跑」；生产仍可配合 `deploy/systemd` 拉起 scheduler 进程。

---

## 运维命令

| 命令 | 检查项 |
|------|--------|
| `geegoo doctor` | config、mcp_token、MCP/Signal/Data/Runtime 健康、checkTradingDay 抽样 |
| `geegoo verify --codes …` | 当日盘前报告字段矩阵（reason≥80、evidence_refs、枚举） |
| `geegoo migrate` | `state/` JSON → SQLite `~/.geegoo/data/agent.db` |

部署见 [cross-cutting/deployment.md](./cross-cutting/deployment.md) 与 `scripts/deploy_agent_server.py`。

---

## 与 Hermes 对照

| Hermes | GeeGooAgent |
|--------|-------------|
| `cli.py` + Gateway 20 平台 | `geegoo chat` + HTTP Runtime（无 IM Gateway） |
| `hermes gateway run` | 不需要；股票场景用 Scheduler + Workflow |
| `batch_runner.py` | `geegoo run` + workflow（确定性步骤为主） |
| ACP / IDE 集成 | 未实现（按需 Phase+） |
