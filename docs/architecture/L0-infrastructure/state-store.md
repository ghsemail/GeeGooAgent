# L0 — StateStore

## 职责

统一持久化 Agent 状态——Runtime 崩溃后恢复依赖此处。

## 存储实体


| Key 空间                   | 内容                 |
| ------------------------ | ------------------ |
| `session/{id}`           | Session JSON       |
| `working/{id}`           | WorkingMemory JSON |
| `checkpoint/{id}/{step}` | Checkpoint 快照      |
| `metrics/{id}`           | Cost/tracing 汇总    |


## 接口

```python
class StateStore(Protocol):
    def save(self, key: str, data: dict) -> None: ...
    def load(self, key: str) -> dict | None: ...
    def list_keys(self, prefix: str) -> list[str]: ...
    def delete(self, key: str) -> None: ...

class FileStateStore(StateStore):
    def __init__(self, root: Path): ...
```

## 路径布局

```text
{output_dir}/
  20260605/
    sessions/sess-abc.json
    working/sess-abc.json
    checkpoints/sess-abc/step-087.json
    artifacts/sess-abc/...
```

## 后期

`SQLiteStateStore` — 多用户查询、索引 session 状态。

## MVP

FileStateStore only。

## 代码

`src/geegoo/infra/state_store.py`