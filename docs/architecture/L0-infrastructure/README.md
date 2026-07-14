# L0 — Infrastructure Layer

横切基础设施。决定 Agent 能否**长时运行、可恢复、可观测**。

```text
Agent = Runtime + Infrastructure
```

## 模块设计说明

L0 是 Agent 的**操作系统底座**：不承载业务逻辑，但为 L4 Runtime 提供进程级能力。设计公式中 Infrastructure 与 Runtime 并列，因为缺少 L0 时 Agent 只能跑「一次性脚本」，无法 cron 恢复、无法审计、无法在失败后续跑。

**核心设计决策**

| 决策                            | 理由                                                          |
| ----------------------------- | ----------------------------------------------------------- |
| 事件驱动（EventBus）                | Runtime、Logging、Supervisor、Scheduler 解耦；Loop 只发事件，不直接调日志/告警 |
| 状态外置（StateStore + Checkpoint） | Session/Working 每步落盘；进程崩溃后可从最近 checkpoint 恢复                |
| 调度外置（Scheduler）               | MVP 用 systemd timer 触发 CLI；Agent 本身不内置 cron 守护              |
| Sandbox 在 L0 而非 L2            | 工作区边界、网络 allowlist、资源限额是**环境策略**，Tools 只声明需求                |
| 同步 InProcess EventBus（MVP）    | 单进程盘前任务足够；后期可换 AsyncEventBus                                |

**模块协作**

```text
Scheduler ──RunRequested──▶ Runtime
Runtime ──ToolCompleted──▶ Checkpoint ──▶ StateStore
Runtime ──*──▶ EventBus ──▶ Logging / Tracing / Supervisor
Tools ──▶ SandboxManager（策略校验）──▶ HTTP allowlist
Secrets ──▶ Clients / Gateway（凭证注入，禁止进 git）
```

**边界**

- **提供**：事件总线、持久化、检查点、定时触发、沙箱策略、日志/追踪、密钥、环境 profile
- **不提供**：LLM 调用、Tool 注册、Skill 加载、业务解析（周线趋势、报告渲染等归 L4/L5）
- **被谁使用**：L4 Runtime（主消费者）、L2 Tools（Sandbox/Secrets）、L5 CLI（Scheduler 入口）

**MVP 范围**

Phase 0 四件套必做：`EventBus` → `StateStore` → `Checkpoint` → `Scheduler`。Sandbox 六层与 Logging 随 Phase 1 盘前一并落地；Timer/Tracing 可 stub。

## Phase 0 四件套（P0）

| 模块         | 文档                                 | MVP             |
| ---------- | ---------------------------------- | --------------- |
| EventBus   | [event-bus.md](./event-bus.md)     | 同步 in-process   |
| StateStore | [state-store.md](./state-store.md) | FileStateStore  |
| Checkpoint | [checkpoint.md](./checkpoint.md)   | 每步保存            |
| Scheduler  | [scheduler.md](./scheduler.md)     | systemd adapter |

## 其余模块

| 模块         | 文档                                 | MVP               |
| ---------- | ---------------------------------- | ----------------- |
| Timer      | [timer.md](./timer.md)             | stub              |
| Sandbox    | [sandbox.md](./sandbox.md)         | SandboxManager 六层 |
| Logging    | [logging.md](./logging.md)         | 双轨                |
| Tracing    | [tracing.md](./tracing.md)         | StepRecord        |
| Secrets    | [secrets.md](./secrets.md)         | File+Env          |
| EnvManager | [env-manager.md](./env-manager.md) | profile           |

## 代码包

`src/geegoo/infra/`

## 依赖规则

- 所有上层可使用 infra
- infra **不得**依赖 runtime / tools / llm

