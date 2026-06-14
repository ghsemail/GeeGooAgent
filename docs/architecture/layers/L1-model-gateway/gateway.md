# L1 — Model Gateway

## 职责

```text
Planner → Gateway.chat() → Provider → 模型
```

- 统一 `chat(messages, tools) -> LLMResponse`
- Fallback、重试、限流（轻量）
- 委托 CostManager 记 token

## 接口

```python
class ModelGateway:
    def __init__(
        self,
        primary: LLMProvider,
        fallback: LLMProvider | None,
        cost: CostManager,
        config: GatewayConfig,
    ): ...

    def chat(self, messages: list[Message], tools: list[ToolSchema]) -> LLMResponse: ...
```

## 失败策略

1. 主模型失败 → 同模型重试 3 次（间隔 5s）
2. 仍失败 → 若有 fallback，切换模型 1 次
3. 仍失败 → 抛出 `ModelGatewayError` → Runtime `RunFailed`

## 配置

```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4o",
    "api_key_env": "OPENAI_API_KEY"
  },
  "llm_fallback": {
    "provider": "anthropic",
    "model": "claude-sonnet-4-20250514",
    "api_key_env": "ANTHROPIC_API_KEY"
  }
}
```

## MVP

单主 + 可选 fallback；无限流缓存。