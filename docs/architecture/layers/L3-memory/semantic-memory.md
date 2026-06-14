# L3 — SemanticMemory

## 职责

向量检索历史报告与相似行情 setup（Phase 4+）。

## 规划

- 对 `{code}-premarket.md` chunk + embedding
- Tool：`recall_similar_setup(code, market_condition)`

## 前置条件

仅在 Phase 4+ 且产品需要 `recall_similar_setup` 时实现。MVP 与 Phase 2–3 **均不需要**向量库或 embedding——见 [README.md §外部依赖决策](./README.md#外部依赖决策数据库--向量库--embedding)。

## MVP

**不实现**。接口 stub：

```python
class SemanticMemory:
    def search(self, query: str, k: int = 3) -> list[Chunk]: ...
    # MVP: return []
```

