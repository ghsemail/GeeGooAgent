# L1 — Cost Manager

## 职责

统计每次 LLM 调用的 token 与估算费用，写入 `metrics.json`。

## 接口

```python
@dataclass
class TokenUsage:
    prompt_tokens: int
    completion_tokens: int
    model: str
    estimated_usd: float

class CostManager:
    def record(self, session_id: str, step: int, usage: TokenUsage) -> None: ...
    def session_total(self, session_id: str) -> TokenUsage: ...
```

## 单价表

配置在 `config.cost_rates` 或常量（按 model 名匹配）。

## 输出

`{output_dir}/{date}/metrics.json`：

```json
{
  "session_id": "sess-abc",
  "steps": [{"step": 1, "model": "gpt-4o", "prompt": 1200, "completion": 400, "usd": 0.008}],
  "total_usd": 0.32
}
```

## MVP

记录 token；USD 估算可选。