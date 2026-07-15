# L0 — Timer

## 职责

- `sleep(duration)` — 等待后继续
- `schedule(callback, run_after)` — 延迟执行
- 盘中轮询、webhook 等待、重试退避

## 接口

```python
class Timer(Protocol):
    def sleep(self, seconds: float) -> None: ...
    def schedule(self, fn: Callable, delay: timedelta) -> str: ...
    def cancel(self, timer_id: str) -> None: ...
```

## MVP

```python
class SyncTimer:
    def sleep(self, s): time.sleep(s)
    def schedule(...): raise NotImplementedError  # Phase 3
```

盘前定时由 **OS systemd** 承担，应用内 Timer 可 stub。

## Phase 3 用例

- 盘中每 5min 轮询持仓
- webhook 触发后 `await sleep(300)` 再拉报告

## 代码

`src/geegoo/infra/timer.py`