# L5 — 触发入口

## 职责

将外部事件转换为 `AgentRuntime.run(skill, context)` 调用。

## 触发器类型

| 类型             | 实现                       | Phase | 文档                                                                       |
| -------------- | ------------------------ | ----- | ------------------------------------------------------------------------ |
| **CLI**        | `geegoo-agent run <skill>` | 1     | 本文                                                                       |
| **Timer (OS)** | systemd `.timer`         | 1     | [cross-cutting/deployment.md](../../cross-cutting/deployment.md)         |
| **Webhook**    | `server/webhook.py`      | 3     | —                                                                        |
| **Chat**       | `geegoo-agent chat` REPL   | 4     | —                                                                        |
| **Resume**     | `geegoo-agent resume <id>` | 1     | [L0-infrastructure/checkpoint.md](../../L0-infrastructure/checkpoint.md) |

## CLI 命令表

| 命令                                          | 说明              |
| ------------------------------------------- | --------------- |
| `geegoo-agent run pre_market`                 | 新建 Session      |
| `geegoo-agent run pre_market --dry-run`       | 不写 GeeGoo API     |
| `geegoo-agent run pre_market --code 00700.HK` | 单股              |
| `geegoo-agent run pre_market --force`         | 忽略幂等            |
| `geegoo-agent resume <session_id>`            | 从 Checkpoint 恢复 |
| `geegoo-agent session list`                   | 列出 Session      |
| `geegoo-agent tools check`                    | API 冒烟          |

## RunContext

```python
@dataclass
class RunContext:
    skill_name: str
    mode: RunMode              # scheduled | interactive | signal
    trigger_source: str        # "systemd" | "cli" | "webhook"
    dry_run: bool = False
    force: bool = False
    stock_filter: str | None = None
    session_id: str | None = None   # resume 时
```

## 与 Scheduler 关系

- MVP：systemd 调 CLI，`Scheduler` 仅文档 + adapter 接口
- Phase 3：`EmbeddedScheduler` 处理 webhook 排队

## 发布事件

```python
bus.emit("RunRequested", {"skill": "pre_market", "mode": "scheduled"})
# ... 结束时
bus.emit("RunFinished", {"session_id": "...", "status": "completed"})
```

