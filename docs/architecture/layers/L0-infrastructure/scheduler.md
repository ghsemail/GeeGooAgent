# L0 — Scheduler

## 职责

按 cron 触发 Skill Workflow，并根据 Supervisor verdict 退避重试。

## 实现（Go）

| 组件 | 路径 |
|------|------|
| Cron 引擎 | `internal/scheduler/scheduler.go`（robfig/cron/v3） |
| 任务定义 | `internal/scheduler/jobs.go` — `jobs.json` |
| CLI | `geegoo scheduler run` / `geegoo scheduler list` |

### 默认任务

`DefaultJobs()` 含 `pre_market` 交易日 08:00（可配置 `jobs.json`）。

### 运行语义

1. Tick → `App.RunSkill(skillName)`
2. 读取 workflow Supervisor verdict：`pass` / `recoverable` / `terminal`
3. `recoverable` 可配置退避后重跑；`terminal` 记录告警

### 与 systemd

- **推荐**：`geegoo scheduler run` 长驻（SIGTERM 优雅退出）
- **可选**：systemd 仅拉起 scheduler 进程，而非每个 skill 单独 timer

## 配置示例

`~/.geegoo/jobs.json`：

```json
[
  {
    "name": "pre_market",
    "skill": "pre_market",
    "cron": "0 8 * * 1-5",
    "enabled": true
  }
]
```

## 未实现

- 通用内存 TaskQueue / 多 worker 队列
- Webhook 触发排队（Phase 3+）
- Bot 侧自动交易调度

## 相关

- [entrypoints.md](../../entrypoints.md)
- [../L4-runtime/workflow-engine.md](../L4-runtime/workflow-engine.md)
