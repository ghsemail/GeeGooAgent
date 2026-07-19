# Agent Loop 对标

> **部署记录**：2026-07-19，`225615ed` 已上线 GeeGooAgent 节点（119.45.16.112）。`geegoo doctor` 全绿；`geegoo inspect --quick` 9/9 PASS。

本目录聚焦 **Agent Loop**（单轮或多轮「观察 → 规划 → 行动 → 更新」循环）的实现差异，不覆盖 IM 网关、盘前 Workflow、TUI 皮肤等外围能力。

| 文档 | 对比轴 | 说明 |
|------|--------|------|
| **[optimization-roadmap.md](./optimization-roadmap.md)** | **优化方案** | 借鉴 Hermes / Grok Build 的分阶段 Agent Loop 路线图 |
| [hermes.md](./hermes.md) | **GeeGooAgent × Hermes Agent** | 金融 Agent 从 Hermes cron 迁移后的 loop 对齐度、双轨编排、优劣 |
| [grok-build.md](./grok-build.md) | **GeeGooAgent × Grok Build** | 编码 harness（`xai-grok-shell`）与 GeeGoo `internal/agent` 的 loop 机制对比 |

## 阅读顺序

1. 若关心 **GeeGoo 与 Hermes 是否对齐、盘前为何不用纯 ReAct** → [hermes.md](./hermes.md)
2. 若需要 **落地优化计划** → [optimization-roadmap.md](./optimization-roadmap.md)
3. 若关心 **Plan mode、并行子 Agent、Headless CI** 等编码 harness 能力 → [grok-build.md](./grok-build.md)
4. 三方全维度速查表 → [../comparison.md](../comparison.md)
5. GeeGoo loop 实现细节 → [../../architecture/layers/L4-runtime/agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md)
6. **功能怎么测** → [../../architecture/layers/L4-runtime/agent-loop-verification.md](../../architecture/layers/L4-runtime/agent-loop-verification.md)
7. Hermes P1–P8 交付记录 → [../../../deploy/hermes-parity-comparison.md](../../../deploy/hermes-parity-comparison.md)

## 图例

| 符号 | 含义 |
|------|------|
| ✅ | 已具备且生产可用 |
| ⚠️ | 部分实现、降级或场景受限 |
| ❌ | 未实现 / 明确不做 |
| — | 不适用（非该产品 loop 目标） |

## 代码锚点（GeeGooAgent）

| 组件 | 路径 |
|------|------|
| 对外入口 | `internal/agent/agent.go` — `Agent.Run` |
| ReAct 循环 | `internal/agent/loop.go` — `Loop.RunTurn` |
| 单轮迭代 | `internal/agent/loop_round.go` |
| 预算耗尽 | `internal/agent/loop_budget.go` |
| 上下文压缩 | `internal/agent/loop_compress.go` + `internal/prompt/compressor.go` |
| 子 Agent | `internal/agent/subagent.go` + `delegate_tool.go`（`delegate_task` / `delegate_tasks`） |
| 工具 schema | `internal/tools/schema_validate.go` |
| Hooks | `internal/tools/hooks.go` |
| Loop 自检 | `cmd/geegoo/inspect.go` → `internal/inspect/report.go` |
| NDJSON 事件 | `internal/runtime/agent_events.go` |
| HTTP clarify | `internal/runtimeapi/clarify_hub.go` |
| 离线验收 | `geegoo verify agent-loop` → `internal/verify/agent_loop.go` |

## 外部参考

- Hermes Agent 架构：<https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture>
- Grok Build 仓库：<https://github.com/xai-org/grok-build>
- Grok Build 文档：<https://docs.x.ai/build/overview>
