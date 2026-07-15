# 自托管 Agent 平台 — 通用设计蓝图

> **Canonical 通用蓝图**。面向「从零生成一个自托管 Agent」的智能体与人类开发者。  
> **参考实现**：[GeeGooAgent](../../../README.md)（Go 单二进制 `geegoo`）。  
> **GeeGoo 领域细节**见上级 [../README.md](../README.md) 与 [../domains/](../domains/)。

---

## 本蓝图回答什么

| 问题 | 文档 |
|------|------|
| 要建什么、不建什么？ | [../overview.md](../overview.md) |
| 目录与包怎么划？ | [repo-layout.md](./repo-layout.md) |
| 六层各自接口与边界？ | [layers.md](./layers.md) |
| Skill 怎么扩展业务？ | [skill-pack.md](./skill-pack.md) |
| 分几期交付、验收标准？ | [phases.md](./phases.md) |
| **智能体如何按步实现？** | [agent-build-guide.md](./agent-build-guide.md) |

---

## 设计公式（不可改）

```text
Agent = Agent Runtime (L4) + Infrastructure (L0) + Skill Pack (L5)
```

- **Runtime**：ReAct Loop（交互）+ WorkflowEngine（确定性批量）
- **Infrastructure**：EventBus、StateStore、Checkpoint、Scheduler 适配
- **Skill Pack**：manifest 声明 Tool 白名单与 workflow，业务与平台解耦

---

## 依赖方向（实现时不得违反）

```text
L5 Application  →  CLI / Skill / Rules
L4 Runtime      →  ReAct / Workflow / Session
L3 Memory       →  Working / Session
L2 Tools        →  Registry / Clients / Catalog
L1 Gateway      →  LLM Provider 抽象
L0 Infra        →  持久化 / 事件 / 调度 / 沙箱
```

**铁律**

1. LLM 编排，Tool 执行 — Runtime **禁止**直连 HTTP / 任意 Shell
2. Runtime **禁止**直连 LLM — 必须经 L1 Gateway
3. 长流程 **每步 Checkpoint** — 崩溃可 `resume`
4. Working Memory **结构化 Apply** — 大 payload 不进 prompt
5. 定时模式 **过滤危险 Tool** — 无人值守禁 mutating CRUD

---

## 智能体阅读顺序（生成新 Agent 前必读）

1. [../overview.md](../overview.md) — 定位、六层、外部依赖
2. [repo-layout.md](./repo-layout.md) — 必须创建的文件树
3. [layers.md](./layers.md) — 每层接口伪代码 + MVP 边界
4. [skill-pack.md](./skill-pack.md) — manifest / supervisor_checks 规范
5. [phases.md](./phases.md) — Phase 0→3 交付物与验收
6. [agent-build-guide.md](./agent-build-guide.md) — **15 Step 复制即用指令**

---

## 与 GeeGoo Agent 的关系

| 维度 | 通用蓝图 | GeeGoo Agent |
|------|----------|--------------|
| 分层 | L5→L0 六层 | 同左 |
| 语言 | 推荐 Go 或 Python ≥3.11 | **已实现 Go** |
| 首个 Skill | 由你定义 | `pre_market` |
| 外部 API | 抽象 Client | GeeGoo MCP 3120 等 |
| 工程细则 | 本目录 + agent-build-guide | [engineering/](../../engineering/) |

实现 GeeGoo 专用 Agent：本蓝图 + [engineering/requirements.md](../../engineering/requirements.md) + [domains/](../domains/)。

实现**任意领域**自托管 Agent：只读本目录 + 自定义 `skills/<name>/` 与 Tool Catalog。

---

## MVP 一句话

**Phase 0**：空壳 CLI（setup / doctor / chat）+ L0 四件套 + Registry + 1 mock Tool + ReAct 单轮。  
**Phase 1**：第一个 Skill 的 Workflow 端到端 + checkpoint/resume + dry-run。  
**Phase 2**：ReAct chat 接 Skill tool 白名单 + Session 持久化。  
**Phase 3**：SkillLoader 读 manifest + Supervisor 读 yaml 验收。

详见 [phases.md](./phases.md)。
