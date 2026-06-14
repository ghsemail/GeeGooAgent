# L3 — EpisodicMemory

## 职责

跨 Run 的**情节记忆**：历史日志、报告、态度轨迹。

## 数据源

| 路径 | 用途 |
|------|------|
| `{date}/execution-log.md` | 排错、审计 |
| `{date}/{code}-premarket.md` | 昨日报告召回 |
| `attitude_history.jsonl` | 态度趋势 |

## recall Tools

| Tool | Phase |
|------|-------|
| `recall_yesterday_summary(code)` | MVP |
| `recall_past_attitude(code, days)` | Phase 2 |

## 接口

```python
class EpisodicMemory:
    def get_yesterday_report_summary(self, code: str, date: date) -> str: ...
    def append_attitude(self, bot_id: str, attitude: str, date: date) -> None: ...
```

## MVP

`recall_yesterday_summary` 读本地归档；无向量检索。
