# L4 — Planner

## 职责

- 组装 LLM 输入（system + messages + working summary）
- 调用 **L1 Model Gateway**（禁止直连 provider）
- 解析 `tool_calls` 或结束信号

## 接口

```python
class Planner:
    def __init__(self, gateway: ModelGateway, context_builder: ContextBuilder): ...

    def plan(
        self,
        session: Session,
        working: WorkingMemory,
        tools: list[ToolSchema],
    ) -> PlanResult: ...

@dataclass
class PlanResult:
    reasoning: str | None
    tool_calls: list[ToolCall]
    done: bool
    raw_usage: TokenUsage
```

## Context 组成

```text
1. system prompt（SkillLoader 合并结果）
2. working_summary（~500 tokens，结构化进度）
3. 最近 N 轮 messages（含 tool 摘要，非全文）
4. （可选）episodic 召回片段
```

## 与 L2 GeeGoo Analysis LLM 分工


|     | L1 Planner 用的 Gateway | GeeGoo getMCPAnalysis     |
| --- | --------------------- | ----------------------- |
| 用途  | 编排、综合、写报告             | 技术面深度分析                 |
| 调用方 | Planner               | `get_mcp_analysis` Tool |


Planner **不应**在文本里重复生成完整技术分析，应读 Tool 返回的 `analysis_result`。

## 失败处理

- Gateway 超时 → 重试 3 次（Gateway 内）
- 仍失败 → `bus.emit("RunFailed")` + Session failed
- 可选 fallback 模型（Anthropic ↔ OpenAI）

## MVP

OpenAI 或 Anthropic 单主模型 + 可选 fallback。