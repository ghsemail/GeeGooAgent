# L3 — Context Compaction

## 职责

防止 SessionMemory 超过 `context_token_budget`。

## 四级策略


| 级别      | 触发                 | 动作                                   |
| ------- | ------------------ | ------------------------------------ |
| L1 工具截断 | 每个 tool result 写入时 | 全文→Working；Session 留 500 token 摘要    |
| L2 滑动窗口 | token > 60k        | 保留 system + 最近 8 轮 + Working Summary |
| L3 阶段压缩 | phase_a → phase_b  | 插入 PhaseASummary，删 5 指数原始 results    |
| L4 紧急压缩 | token > 90k        | LLM 生成滚动摘要；archive 旧 messages        |


## 接口

```python
class ContextBuilder:
    def build(self, session, working) -> list[Message]: ...
    def compact_if_needed(self, session) -> None: ...
```

## MVP

L1 + L2；L3 规则生成摘要（可不调 LLM）。