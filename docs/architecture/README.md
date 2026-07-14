# GeeGooAgent 架构文档

> 本目录是 GeeGooAgent 的**权威设计文档**。实现以 `internal/` Go 代码为准；文档描述「为什么这样分层」与「各模块如何协作」。

## 从哪里开始

| 你是… | 先读 |
|--------|------|
| 第一次接触代码库 | **[overview.md](./overview.md)** — 系统概览、目录结构、数据流、子系统索引 |
| 要查 Tool 能不能用 | **[../reference/geegoo-agent-tools-tree.md](../reference/geegoo-agent-tools-tree.md)** + [tools-and-skills.md](./tools-and-skills.md) |
| 要改 Agent 核心循环 | [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) |
| 要加 Tool / 接 MCP | [layers/L2-tools/README.md](./layers/L2-tools/README.md) |
| 要加 Skill / 工作流 | [layers/L5-application/skills.md](./layers/L5-application/skills.md) |
| 要 fork 新领域 Agent | [platform-blueprint/README.md](./platform-blueprint/README.md) |

对照参考：[Hermes Agent 架构](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture)（GeeGoo 借鉴目录与模块划分，但不实现 IM Gateway / ACP / 插件市场等无关部分）。

---

## 文档结构（像一本书）

### 第一篇 · 总览

| 章 | 文档 | 内容 |
|----|------|------|
| 1 | [overview.md](./overview.md) | **主架构页**：入口、Agent 核心、数据流、设计原则、依赖链 |
| 2 | [00-overview.md](./00-overview.md) | 设计哲学：六层模型、与 Hermes/Claude Code 对照、核心原则 |
| 3 | [repo-layout.md](./repo-layout.md) | 仓库目录与 `internal/` 包对照 |
| 4 | [entrypoints.md](./entrypoints.md) | CLI、HTTP Runtime、Scheduler 入口详解 |

### 第二篇 · 核心子系统（按代码模块）

与 Hermes「主要子系统」章节对应，按**数据流顺序**阅读：

| 序 | 子系统 | 文档 | Go 包 |
|----|--------|------|-------|
| 1 | 入口点 | [entrypoints.md](./entrypoints.md) | `cmd/geegoo`, `cmd/agent-runtime` |
| 2 | Agent 循环 | [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) | `internal/agent`, `internal/runtime` |
| 3 | Prompt 系统 | [layers/L3-memory/compaction.md](./layers/L3-memory/compaction.md) + [layers/L5-application/rules-prompts.md](./layers/L5-application/rules-prompts.md) | `internal/chatprompt`, `internal/prompt` |
| 4 | Provider 网关 | [layers/L1-model-gateway/README.md](./layers/L1-model-gateway/README.md) | `internal/llm` |
| 5 | **Tools** | [tools-and-skills.md](./tools-and-skills.md) · [layers/L2-tools/README.md](./layers/L2-tools/README.md) | `internal/tools` |
| 6 | **Skills** | [tools-and-skills.md](./tools-and-skills.md) · [layers/L5-application/skills.md](./layers/L5-application/skills.md) | `internal/skills`, `skills/` |
| 7 | 会话与记忆 | [layers/L3-memory/README.md](./layers/L3-memory/README.md) | `internal/chatsession`, `internal/memory` |
| 8 | Workflow + 质检 | [layers/L4-runtime/workflow-engine.md](./layers/L4-runtime/workflow-engine.md) | `internal/workflow`, `internal/verify` |
| 9 | Scheduler | [L0-infrastructure/scheduler.md](./L0-infrastructure/scheduler.md) | `internal/scheduler` |
| 10 | 基础设施 | [L0-infrastructure/README.md](./L0-infrastructure/README.md) | `internal/infra` |

### 第三篇 · GeeGoo 领域集成

| 章 | 文档 | 内容 |
|----|------|------|
| — | [domains/README.md](./domains/README.md) | Skill ↔ Tool ↔ GeeGoo API 映射索引 |
| — | [domains/geegoo-api-routing.md](./domains/geegoo-api-routing.md) | 3120/3200/3210/3230 路由 |
| — | [domains/geegoo-agent-skill-mapping.md](./domains/geegoo-agent-skill-mapping.md) | 盘前 workflow 步骤 → Tool |
| — | [../reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md) | MCP HTTP SSOT（73 路由） |

### 第四篇 · 横切与交付

| 章 | 文档 | 内容 |
|----|------|------|
| — | [cross-cutting/README.md](./cross-cutting/README.md) | Supervisor、可观测性、部署 |
| — | [phases/README.md](./phases/README.md) | 分期路线图与当前完成度 |
| — | [../../deploy/hermes-parity-roadmap.md](../../deploy/hermes-parity-roadmap.md) | P1–P8 Hermes 对齐交付记录 |

### 附录 · 通用 Agent 平台蓝图

若要**从零 fork 任意领域的自托管 Agent**（不限股票）：

→ [platform-blueprint/README.md](./platform-blueprint/README.md)

---

## 六层模型（概念地图）

早期蓝图用 L0–L5 描述依赖方向；**当前 Go 实现**将多层合并在 `internal/` 包中，但概念仍有效：

```text
L5 Application    Skill、CLI、触发、Rules
       ↓
L4 Agent Runtime  ReAct Loop、Workflow、Supervisor
       ↓
L3 Memory         Session、Working、Evidence、Compaction
       ↓
L2 Tools          Registry、MCP Clients、Toolsets
       ↓
L1 Model Gateway  Provider、重试、Fallback
       ↓
L0 Infrastructure SQLite、EventBus、Scheduler、Sandbox
```

**依赖规则**：下层不知上层业务；`infra` 不依赖 `runtime` / `tools`。

各层索引：

| 层 | 目录 |
|----|------|
| L5 | [layers/L5-application/](./layers/L5-application/) |
| L4 | [layers/L4-runtime/](./layers/L4-runtime/) |
| L3 | [layers/L3-memory/](./layers/L3-memory/) |
| L2 | [layers/L2-tools/](./layers/L2-tools/) |
| L1 | [layers/L1-model-gateway/](./layers/L1-model-gateway/) |
| L0 | [L0-infrastructure/](./L0-infrastructure/) |

---

## 实现状态速览（2026-07）

| 能力 | 状态 |
|------|------|
| CLI chat + Hermes 风格 UI | ✅ |
| HTTP Runtime `:3400` | ✅ |
| ReAct + 上下文压缩 | ✅ |
| SQLite 会话 + FTS5 + Evidence | ✅ |
| pre_market 确定性 Workflow | ✅ |
| ~82 Tools 注册 | ✅（部分 Stub/Noop，见 tools-tree） |
| 内置 Scheduler（cron） | ✅ |
| intraday / post_market Skill | 📋 占位 |
| `switch_bot` / `wait_for_human` | ❌ 未注册 |
| 新闻 Script runner | ⚠️ skipped |

---

## 与工程文档的关系

- 编码规范：[../engineering/coding-standards.md](../engineering/coding-standards.md)
- 需求与验收：[../engineering/requirements.md](../engineering/requirements.md)
- Cursor 工作流：[../engineering/cursor-workflow.md](../engineering/cursor-workflow.md)
