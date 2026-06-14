# L1 — LLM Providers

## 抽象

```python
class LLMProvider(Protocol):
    def chat(
        self,
        messages: list[Message],
        tools: list[ToolSchema],
        *,
        temperature: float,
        max_tokens: int,
    ) -> LLMResponse: ...
```

## 实现


| 类                 | 文件                      | API 差异            |
| ----------------- | ----------------------- | ----------------- |
| OpenAIProvider    | `openai_provider.py`    | `tool_calls[]`    |
| AnthropicProvider | `anthropic_provider.py` | `tool_use` blocks |


Gateway 负责格式统一为内部 `ToolCall` 列表。

## LLMResponse

```python
@dataclass
class LLMResponse:
    content: str | None
    tool_calls: list[ToolCall]
    usage: TokenUsage
    raw: dict  # 调试
```

## MVP

实现 OpenAI + Anthropic 两家。