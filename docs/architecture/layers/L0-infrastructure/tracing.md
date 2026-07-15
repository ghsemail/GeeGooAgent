# L0 — Tracing

## 职责

Trace 树：`Task → Tool1 → Tool2`，类似 OpenTelemetry 简化版。

## TraceContext

```python
@dataclass
class TraceSpan:
    span_id: str
    parent_id: str | None
    name: str                  # "tool:get_mcp_analysis"
    start_ms: int
    end_ms: int
    attributes: dict

class TraceContext:
    def start_span(self, name: str, parent: str | None = None) -> str: ...
    def end_span(self, span_id: str, status: str) -> None: ...
    def export_tree(self, session_id: str) -> dict: ...
```

## 输出

并入 `metrics.json` 或 `{session_id}-trace.json`。

## MVP

StepRecord 嵌在 Session 内；无 OTel exporter。

## Phase 7

OpenTelemetry SDK 对接。

## 代码

`src/geegoo/infra/tracing.py`