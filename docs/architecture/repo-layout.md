# 代码仓库布局

Go 主分支实现；与架构六层及 [Agent OS 定稿](./agent-runtime-architecture.md) 的包对照如下。

> 系统导图与数据流 → [overview.md](./overview.md)  
> 逻辑包边界与依赖规则 → 定稿 §5；工程验收 → [engineering/agent-runtime-boundaries.md](../engineering/agent-runtime-boundaries.md)

```text
GeeGooAgent/
├── cmd/
│   ├── geegoo/                 # L5 CLI 入口
│   │   ├── main.go             # 子命令分发
│   │   ├── chat.go             # 交互式 chat
│   │   ├── run.go              # geegoo run <skill>
│   │   ├── ops.go              # setup / doctor / resume
│   │   ├── migrate.go          # 文件 → SQLite 迁移
│   │   ├── skills.go           # geegoo skills list
│   │   ├── scheduler.go        # geegoo scheduler
│   │   └── verify.go           # cutover 验收
│   └── agent-runtime/          # L5 HTTP 入口 (:3400)
│
├── internal/
│   ├── app/                    # ★ 依赖组装（LoadFromConfigPath）
│   ├── agent/                  # L4 Kernel（ReAct Loop + ToolExec）
│   ├── cognition/              # L4 策略（Ranker / Evaluator / PlanPolicy / AdvisorClient）
│   ├── runtime/                # L4 Session + Executor + events
│   ├── workflow/               # L4/L5 确定性工作流 + Supervisor
│   ├── memport/                # L3 Memory port 接口
│   ├── memory/                 # L3 Adapter、Working、Evidence
│   ├── chatsession/            # L3 Session SSOT（SQLite + recall 检索）
│   ├── prompt/                 # L3 上下文压缩
│   ├── chatprompt/             # L5 稳定 system prompt
│   ├── llm/                    # L1 Policy + Gateway + Provider
│   ├── tools/                  # L2 Registry + catalog + bespoke
│   ├── clients/mcp/            # L2 GeeGooBot HTTP 客户端
│   ├── search/                 # L2 DuckDuckGo
│   ├── skills/                 # L5 Skill 注册表
│   ├── scheduler/              # L0 内置 cron
│   ├── report/                 # L5 报告 LLM 综合
│   ├── verify/                 # 横切验收
│   ├── archboundaries/         # import 边界检查
│   ├── infra/                  # L0 DB + State + EventBus
│   ├── config/                 # 配置与端点
│   ├── doctor/                 # 连通性检查
│   ├── runtimeapi/             # HTTP handler
│   ├── httpserver/             # mux
│   ├── auth/                   # Bearer 中间件
│   └── cli/
│       ├── chatrepl/           # REPL + 斜杠命令
│       ├── chatui/             # 终端 UI
│       └── chattui/            # Bubble Tea 多会话
│
├── services/
│   └── cognitive/              # 可选 Python Advisor sidecar（默认不部署）
│       ├── advisor_server.py
│       └── README.md
│
├── skills/                     # L5 Skill 资源（manifest + 模板）
│   ├── pre_market/
│   ├── intraday/
│   └── post_market/
├── rules/                      # L5 常驻规则
├── deploy/
│   └── systemd/                # geegoo-agent、geegoo-advisor 等
├── docs/
├── scripts/
│   ├── install.sh
│   ├── check_import_boundaries.go
│   └── deploy_agent_server.py
├── .github/workflows/ci.yml
├── config.example.json
├── start.sh
└── go.mod
```

## 六层 → Go 包对照

| 层 | 职责 | 主要包 |
|----|------|--------|
| **L5** | Skill、CLI、Rules、报告合成 | `cmd/geegoo`, `internal/skills`, `internal/workflow`, `internal/report`, `skills/`, `rules/` |
| **L4** | Kernel、Cognition、Workflow | `internal/agent`, `internal/cognition`, `internal/runtime`, `internal/workflow` |
| **L3** | Session SSOT、Memory port、压缩 | `internal/chatsession`, `internal/memport`, `internal/memory`, `internal/prompt` |
| **L2** | Tool Registry、MCP | `internal/tools`, `internal/clients/mcp`, `internal/search` |
| **L1** | Model Policy + Gateway | `internal/llm`（`policy.go`, `gateway.go`） |
| **L0** | SQLite、EventBus、Scheduler | `internal/infra`, `internal/scheduler` |

## App 组装（`internal/app`）

`LoadFromConfigPath` 关键步骤：

```text
RebuildGateway()
  → Gateway.SetPolicy(ConfigPolicy + ComplexityPolicy)
  → wireChatMemory()     # memory.Adapter + SetMemory
  → wireCognition()      # cognition.Defaults 或 Advisor Bundle
  → wireRecallRanker()   # Adapter.SessionRanker → agent.RankRecallHits
tools.RegisterAll(..., Deps{Memory: ChatMemory})
Agent.SetSubAgent(...)
```

`config.advisor.enabled` 默认 `false`；无 sidecar 时与纯 Go cognition 等价。

## 依赖方向

```text
cmd/geegoo, cmd/agent-runtime
    → internal/app
        → agent, cognition, runtime, workflow, tools, llm,
          memport, memory, chatsession, prompt, infra, config

agent (Kernel) → cognition | runtime | tools | llm | memport | prompt
cognition  ↛ agent | cli | runtimeapi | tools | app
tools      ↛ cognition
memport    ↛ memory | tools | agent
infra      ↛ runtime | tools | llm | agent
```

Recall 排序（不破坏 tools ↛ cognition）：

```text
recall tool → deps.Memory.Recall
  → memory.Adapter.recallSessions
  → SessionRanker → agent.RankRecallHits → cognition.Ranker
```

边界检查：`go run scripts/check_import_boundaries.go`（CI：`.github/workflows/ci.yml`）。

## 工具注册链

```text
app.LoadFromConfigPath
    → tools.RegisterAll(Deps{Memory: ChatMemory, ...})
        → RegisterHTTPFromCatalog (catalog.AllHTTP)
        → RegisterBespokeTools（含 recall → Memory port）
```

早于 `Agent` 首轮对话；新增 Tool 改 catalog 或 bespoke 即可。

## 已移除（Python 时代）

早期 `src/geegoo/*.py` 已废弃。业务 Agent 逻辑仅在 Go `internal/`；`services/cognitive` 仅为可选 Advisor，不拥有 loop。

## 相关文档

- [overview.md](./overview.md) — 系统图与数据流
- [agent-runtime-architecture.md](./agent-runtime-architecture.md) — Agent OS 定稿
- [entrypoints.md](./entrypoints.md) — 各 cmd 子命令
