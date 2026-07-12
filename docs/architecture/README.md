# GeeGoo Agent 架构蓝图

> **Canonical 设计文档目录**。实现时以本目录为准；Cursor Plan 文件仅作任务跟踪。

## 当前实现总览

👉 **[`overview.md`](overview.md)** — P1–P8 完成后的当前架构（Go 实现）：系统概览图、目录结构、数据流、主要子系统、设计原则、文件依赖链。**新读者从这里开始。**

> 下面的「模块设计说明」是早期 Python 时代的分层蓝图（L0–L5），保留作历史参考；当前 Go 实现以 `overview.md` 为准。

## 模块设计说明（早期蓝图，历史参考）

本目录是 GeeGoo Agent 的**唯一权威设计源**，将「自托管股票分析 Agent」拆为可实现的工程六层 + 横切能力 + 领域映射 + 分期路线。

**设计目标**

- 用 **Tool-Calling Agent**（非 pipeline）替代 Hermes cron：LLM 编排、Typed Tool 执行、显式 ReAct 循环
- 用 **L0 Infrastructure** 支撑长时运行：事件驱动、可恢复、可调度、可观测
- 用 **Skill Pack** 表达业务：盘前/盘后/盘中、按需分析、策略、Bot 管理分装加载

**文档组织原则**

| 类型  | 目录               | 回答的问题                            |
| --- | ---------------- | -------------------------------- |
| 总览  | `00-overview.md` | 为什么这样分层、与 Hermes/Claude Code 的差异 |
| 分层  | `layers/L5`…`L0` | 每层职责、接口、代码包、MVP 边界               |
| 横切  | `cross-cutting/` | 部署、质检、可观测性如何贯穿各层                 |
| 领域  | `domains/`       | GeeGoo 双端口 API、Skill → Tool 映射     |
| 交付  | `phases/`        | 先做什么、后做什么                        |

**依赖方向（实现时不得违反）**

```text
L5 → L4 → L3 → L2 → L1 → L0
```

- 下层不知道上层业务；`infra` 不依赖 `runtime` / `tools`
- Runtime 不直连 LLM、不手写 HTTP；一切外部 IO 经 L2 ToolRegistry

**与代码的关系**

`repo-layout.md` 定义 `src/geegoo/` 与六层一一对应。文档先行、实现跟随时以本目录 diff 为准更新蓝图。

## 阅读顺序

**若要开始写代码**，请先读 [../engineering/requirements.md](../engineering/requirements.md) 与 [../engineering/cursor-workflow.md](../engineering/cursor-workflow.md)。

1. [00-overview.md](./00-overview.md) — 定位、工程六层、核心原则
2. 按层深入（L5 → L0）：
3. [cross-cutting/](./cross-cutting/) — Supervisor、部署、可观测性
4. [domains/](./domains/) — GeeGoo API、双 Skill 映射
5. [phases/](./phases/) — MVP 与分期交付（[roadmap.md](./phases/roadmap.md)）

## 六层索引

| 层                     | 目录                                                     | 职责                                                                        |
| --------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------- |
| **L5 Application**    | [layers/L5-application/](./layers/L5-application/)     | Skill、触发入口、Rules/Prompts                                                  |
| **L4 Agent Runtime**  | [layers/L4-runtime/](./layers/L4-runtime/)             | Planner、Executor、StateMachine、WorkflowEngine                              |
| **L3 Memory**         | [layers/L3-memory/](./layers/L3-memory/)               | 四层记忆与压缩                                                                   |
| **L2 Tools**          | [layers/L2-tools/](./layers/L2-tools/)                 | **~87 Tool**（MVP 19）；[tool-catalog.md](./layers/L2-tools/tool-catalog.md) |
| **L1 Model Gateway**  | [layers/L1-model-gateway/](./layers/L1-model-gateway/) | 模型路由、Fallback、Cost                                                        |
| **L0 Infrastructure** | [L0-infrastructure/](./L0-infrastructure/)             | EventBus、Checkpoint、Scheduler 等                                           |

## 代码包对照

见 [repo-layout.md](./repo-layout.md)。

## MVP 范围（不变）

- **Phase 1**：仅 `skills/pre_market` 盘前端到端
- **Phase 0**：L0 四件套 + L4 Runtime + L1 Gateway 轻量版

## 设计公式

```text
Agent = Agent Runtime (L4) + Infrastructure (L0)
```

实现优先级：**EventBus → StateStore → Checkpoint → Scheduler**（见 [L0-infrastructure/README.md](./L0-infrastructure/README.md)）。