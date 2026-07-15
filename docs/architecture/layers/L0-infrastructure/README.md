# L0 — Infrastructure Layer

横切基础设施：持久化、调度、事件、沙箱策略。

> Go 实现：`internal/infra`、`internal/scheduler`

```text
Agent = Runtime + Infrastructure
```

## Phase 0 四件套

| 模块 | 文档 | Go 实现 | 状态 |
|------|------|---------|------|
| EventBus | [event-bus.md](./event-bus.md) | `infra/events.go` | ✅ 同步 |
| StateStore | [state-store.md](./state-store.md) | `infra/state.go` + SQLite | ✅ |
| Checkpoint | [checkpoint.md](./checkpoint.md) | workflow + `checkpoints` 表 | ✅ |
| Scheduler | [scheduler.md](./scheduler.md) | `internal/scheduler` | ✅ 内置 cron |

## 其余模块

| 模块 | 文档 | 状态 |
|------|------|------|
| SQLite DB | `infra/db.go`, `schema.sql` | ✅ WAL + 7 表 + FTS5 |
| Sandbox / WorkspaceGuard | [sandbox.md](./sandbox.md) | ✅ 路径边界 |
| Logging | [logging.md](./logging.md) | 文件 + execution_log |
| Tracing | [tracing.md](./tracing.md) | 部分 |
| Secrets | [secrets.md](./secrets.md) | config.json + env |
| Timer | [timer.md](./timer.md) | stub |

## 模块协作

```text
geegoo scheduler ──▶ App.RunSkill
workflow.Runner ──▶ Checkpoint ──▶ SQLite
workflow.Runner ──▶ EventBus ──▶ （日志订阅方）
tools ──▶ WorkspaceGuard ──▶ save_local_report 路径校验
```

## Scheduler（重要变更）

早期蓝图：仅 systemd timer 触发 CLI。

**当前**：`geegoo scheduler run` 内置 robfig/cron，支持 supervisor verdict 退避重试。生产仍可用 systemd 拉起 scheduler 进程。

## SQLite 作为统一地基

| 数据 | 表 |
|------|-----|
| Chat 会话 | `chat_sessions`, `session_events` |
| 证据链 | `evidence_records` |
| Workflow 进度 | `working_state`, `checkpoints` |

`geegoo migrate` 从文件 JSON 迁移。

## 边界

- **提供**：DB、事件、调度、工作区沙箱
- **不提供**：LLM、Tool、Skill 业务逻辑

## 依赖规则

- 所有上层可使用 `infra`
- `infra` **不得** import `runtime` / `tools` / `llm`

## 延伸阅读

- [../../entrypoints.md](../../entrypoints.md) §Scheduler
- [../../repo-layout.md](../../repo-layout.md)
