# L0 — Checkpoint

## 职责

Step 级快照：**Step87 API 超时 → 从 Step87 恢复**，无需从 Step1 重来。

## 数据结构

```json
{
  "checkpoint_id": "cp-20260605-087",
  "task_id": "sess-abc",
  "step": 87,
  "status": "running",
  "skill": "pre_market",
  "plan_snapshot": "working summary text",
  "memory_snapshot_ref": "working/sess-abc.json",
  "messages_tail_ref": "sessions/sess-abc-archive-087.json",
  "last_tool": "get_mcp_analysis",
  "created_at": "ISO8601"
}
```

## 接口

```python
class CheckpointManager:
    def save(self, session: Session, working: WorkingMemory) -> str: ...
    def load_latest(self, session_id: str) -> Checkpoint | None: ...
    def list(self, session_id: str) -> list[Checkpoint]: ...
```

## 触发时机

**每个 ReAct 步结束后**（Tool 全部完成后）调用 `save()`。

## resume 流程

1. `CheckpointManager.load_latest(session_id)`
2. 恢复 session.step、messages 尾部
3. `StateStore.load(working_ref)`
4. WorkflowEngine 注入补跑 user message（Supervisor 触发时）

## MVP

每步 checkpoint + CLI resume。

## 代码

`src/geegoo/infra/checkpoint.py`