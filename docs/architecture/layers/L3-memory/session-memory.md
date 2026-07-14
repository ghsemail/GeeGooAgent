# L3 — SessionMemory

## 职责

保存当前 Run 的**对话链**：system、assistant 推理、tool_call、tool_result 摘要。

## 数据结构

```python
@dataclass
class Message:
    role: Literal["system", "user", "assistant", "tool"]
    content: str
    tool_call_id: str | None = None
    tool_calls: list[ToolCall] | None = None
```

## 写入规则

| 来源          | 写入内容                                    |
| ----------- | --------------------------------------- |
| SkillLoader | system（仅初始化一次）                          |
| Planner     | assistant + tool_calls                  |
| Executor    | tool role，**摘要**（非 get_mcp_analysis 全文） |

## token 估算

`ContextBuilder` 维护 `session.token_count`；超阈值触发 [compaction.md](./compaction.md)。

## 与 Checkpoint

Checkpoint 保存 `messages` 尾部引用；大段 archive 到 `{session_id}-archive-{step}.json`。

## MVP

append + token 估算 + 滑动窗口截断。