# L4 — Agent Runtime

> 六层 **L4** 文档目录。定稿：[agent-runtime-architecture.md](../../agent-runtime-architecture.md)。

## 文档概述

本目录是 GeeGooAgent **L4（Agent Runtime）** 的专题文档索引，覆盖对话 ReAct 循环、确定性 Workflow、HTTP 交互协议及验收手册。读者为需要修改 `internal/agent`、`internal/workflow`、`internal/runtimeapi` 的开发者；上层入口见 [entrypoints.md](../../entrypoints.md)，全局边界见 [agent-runtime-architecture.md](../../agent-runtime-architecture.md)。

```text
Agent Runtime = ReAct Loop + Workflow Runner + Supervisor + Cognition（策略面）
```

## 文档

| 文档 | 内容 |
|------|------|
| **[agent-loop.md](./agent-loop.md)** | Agent 循环 SSOT：原理、流程、模块、会话状态、配置 |
| [agent-loop-verification.md](./agent-loop-verification.md) | 验收与运维命令 |
| [workflow-engine.md](./workflow-engine.md) | 确定性工作流与 Run 生命周期 |
| [runtime-clarify.md](./runtime-clarify.md) | HTTP clarify / plan 协议 |

## Go 包

`internal/agent` · `internal/cognition` · `internal/runtime` · `internal/workflow` · `internal/app`

## 数据流

```text
Chat:     chatrepl → Agent.Run → Loop.RunTurn → Gateway + ToolExec
Workflow: geegoo run → workflow.Runner → ToolExec + checkpoint
```

## 延伸阅读

- [../../entrypoints.md](../../entrypoints.md)
- [../../repo-layout.md](../../repo-layout.md)
