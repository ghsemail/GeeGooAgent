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
| `run_probe` | `probe_bot_signal_series`（默认附带 `include_key_levels` / series，供 K 线参考线） |
| `fetch_key_levels` | （可选）probe 未带回关键价位时 `get_key_levels` |
| `evaluate_accuracy` | Episode：Strict（至反向信号）+ Path（至下一同向/样本末，含 peak/maxDD） |
| `build_detail` | `diagnose_bot_signal_series` |
| `diag_compose` | LLM 综合判断 |
| `diag_save_kb` | `save_strategy_knowledge` → 信号诊断目录 |
| `summarize` | Markdown 诊断报告（含知识库 id） |

## K 线关键价位参考线

- probe 传 `key_levels_mode=series`（默认 `key_levels_align=daily`），六档与 bars 等长；策略开发页 K 线展示买段 `support_high` / 卖段 `resist_low` 阶梯线（与 Episode 评价解耦，不再做结构熔断截断）。

## 价格分析 Episode 过滤（可选，默认开）

- **Chat API**：`use_price_analysis_episode_filter`；在 `evaluate_accuracy` 阶段按纯价格结构剔除不允许方向的买/卖触发。
- **workflow_card / 诊断 Markdown**：`price_analysis`（5m/60m/日线/周线表 + 支撑阻力 + 综合句）；`price_episode_filter_stats.buy_triggers` / `sell_triggers` 列出**全部**买/卖触发（`masked=true` 为掩码剔除），与 `stripped_*_triggers` 并存。

## 与 playbook 区别

- 本 workflow **串行固定步骤**，不经过 ReAct。
- **不调用** `run_strategy_backtest`。
