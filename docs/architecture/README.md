# GeeGooAgent 架构文档

> 权威设计文档。**实现状态**以 [implementation-status.md](./implementation-status.md) 与 `internal/` 为准。**待办**仅 [backlog.md](./backlog.md)。

## 阅读路径

| 目标 | 文档 |
|------|------|
| 第一次读代码库 | [overview.md](./overview.md) + [repo-layout.md](./repo-layout.md) |
| **Agent Runtime 架构（定稿）** | **[agent-runtime-architecture.md](./agent-runtime-architecture.md)** |
| **哪些已实现 / 未实现** | **[implementation-status.md](./implementation-status.md)** |
| **后续规划（唯一待办）** | **[backlog.md](./backlog.md)** |
| Tool 能否调用 | [layers/L2-tools/tools-status.md](./layers/L2-tools/tools-status.md) |
| 改 Agent 循环 | [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) |
| Loop 验收命令 | [layers/L4-runtime/agent-loop-verification.md](./layers/L4-runtime/agent-loop-verification.md) |
| 改盘前 Workflow | [layers/L4-runtime/workflow-engine.md](./layers/L4-runtime/workflow-engine.md) |
| 入口与命令 | [entrypoints.md](./entrypoints.md) |
| 代码目录 | [repo-layout.md](./repo-layout.md) |
| GeeGoo API 映射 | [domains/](./domains/) |
| Fork 新领域 Agent | [platform-blueprint/](./platform-blueprint/) |

开发过程稿：[../archive/](../archive/)（不含本目录正文）。

---

## 文档树

```text
architecture/
├── README.md                      # 本索引
├── overview.md                    # 六层导图、数据流
├── agent-runtime-architecture.md  # ★ Agent OS 架构定稿
├── implementation-status.md       # ★ 实现状态 SSOT
├── backlog.md                     # ★ 唯一待办清单
├── repo-layout.md                 # 仓库 ↔ internal/ 包
├── entrypoints.md                 # CLI / HTTP / scheduler
├── layers/
│   ├── L0-infrastructure/
│   ├── L1-model-gateway/
│   ├── L2-tools/
│   ├── L3-memory/
│   ├── L4-runtime/
│   └── L5-application/
├── domains/
├── cross-cutting/
└── platform-blueprint/
```

---

## 六层与 Go 包

| 层 | 文档 | 主要 Go 包 |
|----|------|------------|
| L5 | [layers/L5-application/](./layers/L5-application/) | `cmd/geegoo`, `internal/skills`, `skills/` |
| L4 | [layers/L4-runtime/](./layers/L4-runtime/) | `internal/agent`, `internal/cognition`, `internal/runtime`, `internal/workflow` |
| L3 | [layers/L3-memory/](./layers/L3-memory/) | `internal/chatsession`, `internal/memport`, `internal/memory`, `internal/prompt` |
| L2 | [layers/L2-tools/](./layers/L2-tools/) | `internal/tools`, `internal/clients/mcp` |
| L1 | [layers/L1-model-gateway/](./layers/L1-model-gateway/) | `internal/llm` |
| L0 | [layers/L0-infrastructure/](./layers/L0-infrastructure/) | `internal/infra`, `internal/scheduler` |

---

## 速览

| 能力 | 状态 |
|------|------|
| Agent OS（Cognition / Policy / Memory port） | ✅ |
| `pre_market` / `intraday` / `post_market` workflow | ✅ |
| Chat + TUI + 压缩 + recall | ✅ |
| 82 Tools | ✅（部分 ⚠️，见 tools-status） |
| SQLite + Evidence | ✅ |
| 可选 Python Advisor | ✅（默认关） |
| Dashboard / 向量库 | 📋 [backlog.md](./backlog.md) |

---

## 外部参考

- MCP HTTP：[reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md)
- Hermes 对齐：[deploy/hermes-parity-roadmap.md](../../deploy/hermes-parity-roadmap.md)
- 工程规范：[engineering/](../engineering/)
