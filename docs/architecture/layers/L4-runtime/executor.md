# L4 — Executor

## 职责

- 执行 Planner 输出的 `tool_calls`
- 经 **L0 Sandbox** 包裹（白名单、超时）
- 结果写 WorkingMemory + SessionMemory 摘要

## 接口

```python
class Executor:
    def __init__(
        self,
        registry: ToolRegistry,
        sandbox: Sandbox,
        bus: EventBus,
    ): ...

    def execute(
        self,
        call: ToolCall,
        session: Session,
        working: WorkingMemory,
    ) -> ToolResult: ...
```

## 执行流程

```text
1. registry.get(call.name) — 不存在则 ToolNotFoundError
2. sandbox.validate(call) — 白名单 + dry_run 规则
3. tool.run(parsed_input, ToolContext)
4. 截断超大 result → 全文进 working，摘要进 session
5. tracing.record_tool_span(...)
6. 返回 ToolResult
```

## ToolContext

```python
@dataclass
class ToolContext:
    session_id: str
    working: WorkingMemory
    config: AppConfig
    dry_run: bool
    secrets: SecretsProvider
    state_store: StateStore
```

## 并行 Tool Calls

单步 LLM 可返回多个 tool_call（如 5 个指数）：

- MVP：顺序执行（简单）
- Phase 2：Perception/Analysis 类可 `asyncio.gather`

## MVP

顺序执行 + Sandbox 超时 120s（getMCPAnalysis 可能较慢）。