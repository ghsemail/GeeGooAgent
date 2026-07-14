# L0 — Logging

## 职责

生产可排障：**用户输入、模型输出、工具调用、工具结果、最终输出**。

## 双轨设计

| 轨道      | 输出                     | 受众                     |
| ------- | ---------------------- | ---------------------- |
| **结构化** | JSON → stderr/journald | 运维                     |
| **业务**  | `execution-log.md`     | 人读、Supervisor、Episodic |

## 结构化字段

```json
{
  "ts": "ISO8601",
  "level": "INFO",
  "event": "ToolCompleted",
  "session_id": "sess-abc",
  "step": 12,
  "tool": "get_mcp_analysis",
  "latency_ms": 4500,
  "status": "ok"
}
```

## 与 EventBus

Logging 订阅 `ToolCalled`、`ToolCompleted`、`RunFinished` 写结构化日志。

`write_execution_log` Tool 写业务日志（Agent 显式调用，满足 skill 规范）。

## 配置

```python
def setup_logging(profile: str, json: bool = True) -> None: ...
```

## 代码

`src/geegoo/infra/logging.py`