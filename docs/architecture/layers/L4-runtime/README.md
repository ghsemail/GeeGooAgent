# L4 — Agent Runtime

Agent 的心脏：编排 LLM 与 Tool，管理 Session 状态，驱动 Workflow。

```text
Agent Runtime = ReAct Loop + Workflow Runner + Supervisor
```

> **实现语言**：Go（`internal/agent`, `internal/runtime`, `internal/workflow`）。下文「L4」指概念层，非 Python 包名。

## 模块索引

| 模块 | 文档 | Go 代码 | 状态 |
|------|------|---------|------|
| **Agent 循环** | [agent-loop.md](./agent-loop.md) | `internal/agent`, `internal/runtime/react.go` | ✅ |
| ReAct 设计细节 | [react-loop.md](./react-loop.md) | 同上 | ✅ |
| Executor | [executor.md](./executor.md) | `internal/runtime/executor.go` | ✅ |
| Workflow | [workflow-engine.md](./workflow-engine.md) | `internal/workflow/runner.go` | ✅ pre_market |
| Supervisor | `workflow/supervisor.go` | `internal/workflow/supervisor.go` | ✅ |
| StateMachine | [state-machine.md](./state-machine.md) | session status 字段 | 部分 |
| Planner | [planner.md](./planner.md) | 合并在 ReActLoop + Gateway | ✅ |

## 核心组件职责

| 组件 | 职责 |
|------|------|
| `Agent.Run` | 平台无关单轮对话入口 |
| `ReActLoop` | LLM ↔ Tool 迭代直到完成或 max_steps |
| `Executor` | 调 `tools.Registry`，写回 session messages |
| `workflow.Runner` | 确定性步骤：Phase A + PerStock，checkpoint 幂等 |
| `Supervisor` | 跑后 verdict：pass / recoverable / terminal |
| `Context`（prompt） | 压缩、token 估算 |

## 数据流

### Chat（ReAct）

```text
chatrepl → Agent.Run → ReActLoop → Gateway + Registry
```

### Workflow（确定性）

```text
geegoo run pre_market → App.RunSkill → workflow.Runner
    → 每步 Registry.Execute + Working.Apply + checkpoint
    → finishWithSupervisor → report.Synthesizer → create_pre_market_report
```

## 依赖

- **向上**：L5 `cmd/geegoo`、`internal/skills`
- **向下**：L3 `chatsession`/`memory`、L2 `tools`、L1 `llm`、L0 `infra`

## 边界

- **提供**：编排、循环控制、workflow 步骤、质检 verdict
- **不提供**：HTTP 客户端实现、Skill 文本内容、systemd 单元

## 与 Hermes 对照

| Hermes | GeeGooAgent |
|--------|-------------|
| `AIAgent.run_conversation` | `Agent.Run` / `ReActLoop.RunTurn` |
| 隐式 loop | 显式 `max_tool_rounds` |
| Gateway cron 长 prompt | `workflow.Runner` 确定性步骤 + 可选 LLM 综合 |

## 延伸阅读

- [../entrypoints.md](../entrypoints.md)
- [agent-loop.md](./agent-loop.md)
