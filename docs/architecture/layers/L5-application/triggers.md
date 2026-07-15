# L5 — 触发入口

将外部事件转为 `App.RunSkill` 或 `Agent.Run`（chat）。

## 触发器

| 类型 | 命令 / 入口 | 状态 | 文档 |
|------|-------------|------|------|
| **CLI Workflow** | `geegoo run <skill>` | ✅ | [entrypoints.md](../../entrypoints.md) |
| **CLI Chat** | `geegoo chat` | ✅ | [entrypoints.md](../../entrypoints.md) |
| **Scheduler** | `geegoo scheduler run` | ✅ | [../L0-infrastructure/scheduler.md](../L0-infrastructure/scheduler.md) |
| **Resume** | `geegoo resume --session <id>` | ✅ | [../L0-infrastructure/checkpoint.md](../L0-infrastructure/checkpoint.md) |
| **HTTP Runtime** | `POST /v1/chat/completions` | ✅ | [entrypoints.md](../../entrypoints.md) |
| **Webhook** | — | ❌ | 未实现 |
| **OS Timer only** | systemd 拉起 scheduler | ⚠️ | [cross-cutting/deployment.md](../../cross-cutting/deployment.md) |

## 常用 CLI

| 命令 | 说明 |
|------|------|
| `geegoo run pre_market` | 盘前 Workflow |
| `geegoo run pre_market --dry-run` | 不写远端 API |
| `geegoo resume --session <id>` | 从 checkpoint 幂等续跑 |
| `geegoo chat` | ReAct 对话（TTY 默认 TUI） |
| `geegoo verify --codes ...` | 报告字段验收 |
| `geegoo doctor` | 健康检查 |

## Run 模式

| mode | 行为 |
|------|------|
| `scheduled` / workflow | 确定性步骤；写 report 不经 chat ApprovalGate |
| `interactive` / chat | ReAct；`create_/update_/delete_` 需 ApprovalGate |

## 事件

Workflow 关键节点经 `internal/infra/events.go` EventBus 同步派发（进程内）。
