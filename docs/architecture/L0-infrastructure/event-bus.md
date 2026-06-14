# L0 — EventBus

## 职责

解耦 Runtime、Tools、Logging、Supervisor——Agent 本质是**事件驱动**系统。

## 事件类型


| 事件                | 发布者            | 订阅者                | Payload 要点               |
| ----------------- | -------------- | ------------------ | ------------------------ |
| `RunRequested`    | CLI/Scheduler  | Runtime            | skill, mode              |
| `RunStarted`      | WorkflowEngine | Logging, Tracing   | session_id               |
| `PlanCreated`     | Planner        | Tracing            | step                     |
| `ToolCalled`      | Executor       | Logging, Tracing   | tool, args_hash          |
| `ToolCompleted`   | Executor       | Loop, Logging      | tool, status, latency_ms |
| `MemoryUpdated`   | Executor       | （可选）               | working_path             |
| `CheckpointSaved` | Checkpoint     | Logging            | step                     |
| `RunFinished`     | WorkflowEngine | Scheduler, Logging | status                   |
| `RunFailed`       | Runtime        | Scheduler, Alert   | error                    |


## 接口

```python
class EventBus(Protocol):
    def emit(self, event: str, payload: dict) -> None: ...
    def on(self, event: str, handler: Callable[[dict], None]) -> None: ...

class InProcessEventBus:
    """MVP：同步调用 handlers"""
```

## MVP 实现

`InProcessEventBus`；handlers 内不可抛异常（catch + log）。

## 后期

- `AsyncEventBus` + 队列
- Redis pub/sub（多实例）

## 代码

`src/geegoo/infra/events.py`