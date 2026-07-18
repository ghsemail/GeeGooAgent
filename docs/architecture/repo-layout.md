# 代码仓库布局

Go 主分支实现；与架构六层概念的包对照如下。

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
│   ├── agent/                  # L4 平台无关核心（薄封装）
│   ├── app/                    # 依赖组装（LoadFromConfigPath）
│   ├── runtime/                # L4 ReAct Loop + Executor + Session
│   ├── workflow/               # L4/L5 确定性工作流 + Supervisor
│   ├── skills/                 # L5 Skill 注册表
│   ├── scheduler/              # L0 内置 cron
│   ├── chatprompt/             # L5 稳定 system prompt
│   ├── prompt/                 # L3 上下文压缩
│   ├── llm/                    # L1 Gateway + Provider
│   ├── tools/                  # L2 Registry + catalog + bespoke
│   ├── clients/mcp/            # L2 GeeGooBot HTTP 客户端
│   ├── search/                 # L2 DuckDuckGo
│   ├── chatsession/            # L3 SQLite 会话 + recall
│   ├── memory/                 # L3 Working + Evidence
│   ├── report/                 # L5 报告 LLM 综合
│   ├── verify/                 # 横切验收
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
├── skills/                     # L5 Skill 资源（manifest + 模板）
│   ├── pre_market/
│   └── bundled/
├── rules/                      # L5 常驻规则（attitude、报告格式）
├── deploy/                     # systemd、Hermes 对齐文档
├── docs/
│   ├── architecture/           # 本目录
│   ├── reference/            # MCP interface-map、tools-status
│   └── engineering/
├── scripts/
│   ├── install.sh              # 自托管安装
│   └── deploy_agent_server.py  # 119.45.16.112 部署
├── config.example.json
├── start.sh                    # 服务器 build + runtime 管理
└── go.mod
```

## 六层 → Go 包对照

| 层 | 职责 | 主要包 |
|----|------|--------|
| **L5** | Skill、CLI、Rules、报告合成 | `cmd/geegoo`, `internal/skills`, `internal/workflow`, `internal/report`, `skills/`, `rules/` |
| **L4** | ReAct、Workflow、Supervisor | `internal/agent`, `internal/runtime`, `internal/workflow` |
| **L3** | 会话、Working、Evidence、压缩 | `internal/chatsession`, `internal/memory`, `internal/prompt` |
| **L2** | Tool Registry、MCP 客户端 | `internal/tools`, `internal/clients/mcp`, `internal/search` |
| **L1** | LLM Gateway | `internal/llm` |
| **L0** | SQLite、EventBus、Scheduler | `internal/infra`, `internal/scheduler` |

## 依赖方向（只允许向下）

```text
cmd/geegoo, cmd/agent-runtime
    → internal/app
        → agent, runtime, workflow, tools, llm, chatsession, memory, infra
tools → clients/mcp → HTTP
infra 不得 import runtime / tools / llm
```

## 工具注册链

```text
app.LoadFromConfigPath
    → tools.RegisterAll
        → RegisterHTTPFromCatalog (catalog.AllHTTP)
        → RegisterBespokeTools
```

早于任何 `Agent` 实例创建；新增 Tool 只需改 catalog 或 bespoke，无需改入口。

## 已移除（Python 时代）

早期蓝图中的 `src/geegoo/*.py` 已废弃。若文档仍引用 `runtime/loop.py` 等路径，以本文件与 `internal/` 为准。

## 相关文档

- [overview.md](./overview.md) — 带注释的目录树
- [entrypoints.md](./entrypoints.md) — 各 cmd 子命令说明
