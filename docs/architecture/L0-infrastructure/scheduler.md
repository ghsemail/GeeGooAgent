# L0 — Scheduler

## 职责

任务排队、优先级、重试、取消——类似 Celery/Temporal 的**简化版**。

## 接口

```python
@dataclass
class Task:
    id: str
    skill: str
    mode: RunMode
    priority: int = 0
    scheduled_at: datetime | None = None
    retry_count: int = 0

class Scheduler(Protocol):
    def enqueue(self, task: Task) -> str: ...
    def cancel(self, task_id: str) -> bool: ...
    def poll(self) -> Task | None: ...  # worker 模式
```

## MVP 实现

### SystemdScheduler（主路径）

- 不跑常驻 worker
- `deploy/geegoo-agent-pre.timer` → `geegoo-agent run pre_market`
- `Scheduler` 接口仅用于测试 + 未来 EmbeddedScheduler

### TaskQueue stub

内存队列 `enqueue` 记录任务元数据；单进程 CLI 同步执行。

## 与 Timer 区别


|     | Scheduler    | Timer           |
| --- | ------------ | --------------- |
| 职责  | 任务队列、重试、取消   | 延迟/睡眠/轮询        |
| MVP | systemd 触发   | stub            |
| 例子  | 08:00 盘前任务入队 | 等 webhook 30min |


## Phase 3

`EmbeddedScheduler`（APScheduler）处理 webhook 排队与重试。

## 代码

`src/geegoo/infra/scheduler.py`