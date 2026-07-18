# GeeGooAgent 架构文档

> 权威设计文档；**实现状态**以 [implementation-status.md](./implementation-status.md) 与 `internal/` 为准。

## 阅读路径

| 目标 | 文档 |
|------|------|
| 第一次读代码库 | [overview.md](./overview.md) |
| **哪些已实现 / 未实现** | **[implementation-status.md](./implementation-status.md)** |
| Tool 能否调用 | [layers/L2-tools/tools-status.md](./layers/L2-tools/tools-status.md) |
| 改 Agent 循环 | [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) |
| 改盘前 Workflow | [layers/L4-runtime/workflow-engine.md](./layers/L4-runtime/workflow-engine.md) |
| 入口与命令 | [entrypoints.md](./entrypoints.md) |
| 代码目录 | [repo-layout.md](./repo-layout.md) |
| GeeGoo API 映射 | [domains/](./domains/) |
| 业务能力分期 | [phases/README.md](./phases/README.md) |
| Fork 新领域 Agent | [platform-blueprint/](./platform-blueprint/) |

开发过程稿：[../archive/](../archive/)（不含本目录正文）。

---

## 文档树

```text
architecture/
├── README.md                 # 本索引
├── overview.md               # 主架构：系统图、数据流、六层、原则
├── implementation-status.md  # ★ 实现状态 SSOT
├── repo-layout.md            # 仓库 ↔ internal/ 包
├── entrypoints.md            # CLI / HTTP / scheduler
├── phases/                   # 业务能力路线图
├── layers/
│   ├── L0-infrastructure/    # SQLite、EventBus、Scheduler、Sandbox
│   ├── L1-model-gateway/     # LLM Gateway
│   ├── L2-tools/             # 82 Tool；tools-status = 运行态 SSOT
│   ├── L3-memory/            # Session、压缩、Working、Evidence
│   ├── L4-runtime/           # ReAct、Workflow、Supervisor
│   └── L5-application/       # Skill、触发、Rules
├── domains/                  # geegoo Skill ↔ Tool ↔ 端口
├── cross-cutting/            # 部署、可观测性、Supervisor
└── platform-blueprint/       # 通用 Agent 蓝图（fork 用）
```

---

## 六层与 Go 包

| 层 | 文档 | 主要 Go 包 |
|----|------|------------|
| L5 | [layers/L5-application/](./layers/L5-application/) | `cmd/geegoo`, `internal/skills`, `skills/` |
| L4 | [layers/L4-runtime/](./layers/L4-runtime/) | `internal/agent`, `internal/runtime`, `internal/workflow` |
| L3 | [layers/L3-memory/](./layers/L3-memory/) | `internal/chatsession`, `internal/memory`, `internal/prompt` |
| L2 | [layers/L2-tools/](./layers/L2-tools/) | `internal/tools`, `internal/clients/mcp` |
| L1 | [layers/L1-model-gateway/](./layers/L1-model-gateway/) | `internal/llm` |
| L0 | [layers/L0-infrastructure/](./layers/L0-infrastructure/) | `internal/infra`, `internal/scheduler` |

---

## 速览（详见 implementation-status）

| 能力 | 状态 |
|------|------|
| `pre_market` workflow | ✅ |
| Chat + TUI + 压缩 | ✅ |
| 82 Tools | ✅（部分 ⚠️） |
| SQLite + Evidence | ✅ |
| 内置 Scheduler | ✅ |
| `intraday` / `post_market` | 📋 占位 |
| 新闻 script / Episodic recall | ❌ stub |
| Subagents / Cost / Webhook | ❌ |

---

## 外部参考

- MCP HTTP：[reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md)
- Hermes 对齐：[deploy/hermes-parity-roadmap.md](../../deploy/hermes-parity-roadmap.md)
- 工程规范：[engineering/](../engineering/)
