# L5 — Subagent 委派

## 职责

大任务拆分：子 Session 跑子集 Tool，Orchestrator 只收摘要。

## Subagent 清单

| 名称              | 工具子集                | Phase | MVP |
| --------------- | ------------------- | ----- | --- |
| NewsCollector   | fetch_*_news        | 1     | 可选  |
| StockAnalyst    | analysis + report   | 1-2   | 可选  |
| StrategyAdvisor | strategy + loopback | 5     | 否   |
| BotManager      | bot + reminder      | 6     | 否   |

## spawn_subagent Tool

```python
@dataclass
class SpawnInput:
    agent_type: Literal["stock_analyst", "news_collector", ...]
    task_description: str
    context_refs: list[str]   # working 中的路径

@dataclass
class SpawnOutput:
    sub_session_id: str
    summary: str              # 不进 Orchestrator 全文
    status: str
```

限制：子 agent `max_steps=30`；禁止嵌套 spawn。

## MVP

Orchestrator 直调 Tool；`spawn_subagent` 可 stub 返回 not_implemented。