# L3 — WorkingMemory

## 职责

**结构化任务状态**——Supervisor 和 ContextBuilder 的主要数据源。

## 数据模型

```python
@dataclass
class WorkingMemory:
    session_id: str
    skill: str
    phase: Literal["init", "phase_a", "phase_b", "done"]
    is_trading_day: bool | None
    bot_codes: list[BotStock]
    market_context: MarketContext
    stocks: dict[str, StockWorkspace]
    artifacts: dict[str, str]

@dataclass
class StockWorkspace:
    code: str
    status: Literal["pending", "collecting", "synthesizing", "reported", "skipped", "failed"]
    weekly_analysis_ref: str | None
    attitude: Attitude | None
    capital_distribution: str | None
    stock_news: list[NewsItem]
    synthesis: SynthesisResult | None
```

## Working Summary（注入 Planner）

每步生成 ~500 token 文本：

```text
phase=phase_b | trading_day=true | bots=3 | reported=1/3
pending: 00700.HK(collecting), AAPL.US(pending)
```

## Tools

- `read_working_state(path)` — LLM 按需读字段
- WorkingMemory 由 Executor 在 tool 成功后更新，**不暴露为 LLM Tool**

## 原始数据存储

大文本（analysis_result 全文）存：

```text
{output_dir}/{date}/artifacts/{session_id}/{code}-weekly.md
```

Working 只存 `*_ref` 路径。

## MVP

完整 pre_market 字段 + `apply(tool_call, result)` 更新逻辑。