# L3 — EpisodicMemory

## 状态：❌ 未实现

`recall_yesterday_summary` Tool 已注册，但 `bespoke.go` 固定返回 `StatusSkip`（`not implemented`）。无读取本地 `md` / `jsonl` 的逻辑。

## 目标职责（待做）

跨 Run **情节记忆**：历史 execution-log、昨日盘前 md、态度轨迹。

| 数据源（规划） | 用途 |
|----------------|------|
| `reports/{date}/execution-log.md` | 排错、审计 |
| `reports/{date}/{code}-premarket.md` | 昨日报告摘要 |
| `attitude_history.jsonl` | 态度趋势 |

## 与 Session `recall` 的区别

| | `recall`（✅） | `recall_yesterday_summary`（❌） |
|--|----------------|----------------------------------|
| 范围 | 跨 **chat 会话** FTS | 跨 **自然日** 报告文件 |
| 实现 | `chatsession/recall.go` | 未实现 |

## 实现时建议

1. 只读工作区 `reports/` 下归档，不写 GeeGoo API
2. Workflow `pre_market` 可选步骤；失败应 Skip 而非 Terminal
3. 状态变更后更新 [implementation-status.md](../../implementation-status.md)
