---
name: signal_diagnose
description: 信号诊断 Workflow：读取策略 → probe → Episode 评价 → diagnose → LLM 总结 → 写入知识库（不回测）。
trigger_modes: chat
---

# 信号诊断 Workflow

## 触发示例

- 「**诊断 SAR 策略 · 腾讯**」
- 「信号诊断 Macd4H · 00700」

## 阶段

| Phase | 说明 |
| --- | --- |
| `diag_pick` | 解析策略名与标的 |
| `read_strategy` | `resolveStrategyCatalog` 读策略库 rules |
| `resolve_symbol` | `search_code` → 标准 code |
| `run_probe` | `probe_bot_signal_series` |
| `evaluate_accuracy` | 本地 Episode 评价（至下一次反向信号） |
| `build_detail` | `diagnose_bot_signal_series` |
| `diag_compose` | LLM 综合判断 |
| `diag_save_kb` | `save_strategy_knowledge` → 信号诊断目录 |
| `summarize` | Markdown 诊断报告（含知识库 id） |

## 与 playbook 区别

- 本 workflow **串行固定步骤**，不经过 ReAct。
- **不调用** `run_strategy_backtest`。
