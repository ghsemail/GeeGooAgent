# L4 — Agent Runtime

Agent 的心脏：编排 LLM 与 Tool，管理 Session 状态，驱动 Workflow。

```text
Agent Runtime = Planner + Executor + StateMachine + WorkflowEngine
```

## 模块设计说明

L4 是 **Agent 与 Pipeline 的分水岭**：不是写死「先 A 后 B」的脚本，而是由 Planner 在 ReAct 循环中根据 Working 状态决定下一步调哪个 Tool；WorkflowEngine 只负责 Session 生命周期与 Skill 边界，不硬编码业务步骤。

**核心组件职责**


| 组件             | 职责                                                       |
| -------------- | -------------------------------------------------------- |
| WorkflowEngine | 接收触发、加载 Skill、创建 Session、驱动 Loop 启停                      |
| ReActLoop      | Observe → Plan → Act → Update，直到无 tool_calls 或 max_steps |
| Planner        | 调 L1 Gateway，产出 tool_calls 或最终自然语言                       |
| Executor       | 调 L2 Registry，写回 L3 Memory，发 L0 事件                       |
| StateMachine   | Session 状态：running / agent_done / failed / cancelled     |
| ContextBuilder | 拼 messages：identity + rules + skill prompt + memory      |


**核心设计决策**


| 决策                     | 理由                                        |
| ---------------------- | ----------------------------------------- |
| 显式 ReAct，max_steps 硬上限 | 防止 LLM 空转；盘前预估 ~30–50 步                   |
| 每步 Checkpoint          | 与 L0 配合；单股失败不拖垮全局（continue 下一只）           |
| Skill 约束 Tool 集而非步骤    | 业务变更改 Skill/Rules，不改 Loop 代码              |
| 事件发布而非直接日志             | Loop 只 `emit`；Logging/Tracing 订阅 EventBus |


**单次 run 时序**

```text
L5 trigger → WorkflowEngine.run(skill)
    → Session(created)
    → ReActLoop
        loop: ContextBuilder → Gateway.chat → Executor(tools) → Checkpoint
    → Session(agent_done | failed)
    → emit RunFinished
```

**边界**

- **提供**：编排、会话状态、循环控制、上下文组装
- **不提供**：HTTP、模型 SDK、Skill 内容定义、systemd 单元（L0/L5）
- **依赖**：L5 传入 `LoadedSkill`；向下只通过 Gateway/Registry/Memory/Infra 接口

**MVP 范围**

`AgentRuntime` + `ReActLoop` + 基础 StateMachine + `pre_market` 单 Skill；Subagent spawn Phase 2+。

## 模块索引


| 模块             | 文档                                         | 代码                         |
| -------------- | ------------------------------------------ | -------------------------- |
| ReAct 主循环      | [react-loop.md](./react-loop.md)           | `runtime/loop.py`          |
| Planner        | [planner.md](./planner.md)                 | `loop.py` Plan 阶段          |
| Executor       | [executor.md](./executor.md)               | `loop.py` Act 阶段           |
| StateMachine   | [state-machine.md](./state-machine.md)     | `runtime/session.py`       |
| WorkflowEngine | [workflow-engine.md](./workflow-engine.md) | `runtime/agent_runtime.py` |


## 依赖

- **向上**：L5 Application 调用 `AgentRuntime.run()`
- **向下**：L3 Memory、L2 Tools、L1 Gateway、L0 Infra（EventBus、Checkpoint、StateStore）

## 事件订阅

Runtime 应订阅：

- `ToolCompleted` → 继续 Loop 或 Checkpoint
- `RunFailed` → 标记 Session failed

Runtime 应发布：

- `PlanCreated`、`StepCompleted`、`RunFinished`

