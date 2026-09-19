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
| `fetch_key_levels` | （可选）`get_key_levels` / Key Level Engine；由 `workflow_options.signal_diagnose.use_key_level_episode_stop` 或运营台开关控制 |
| `evaluate_accuracy` | Episode：Strict（至反向信号，可选结构截断）+ Path（至下一同向/样本末，含 peak/maxDD） |
| `build_detail` | `diagnose_bot_signal_series` |
| `diag_compose` | LLM 综合判断 |
| `diag_save_kb` | `save_strategy_knowledge` → 信号诊断目录 |
| `summarize` | Markdown 诊断报告（含知识库 id） |

## 结构截断 Episode（可选）

- **默认规则**：买段 K 线 **low 跌破主支撑 low**；卖段 **high 突破主阻力 high** → Strict 段提前结束（`method` 后缀 `+key_break_support_low`）。
- **关键价位**：probe 传 `key_levels_mode=series`（默认 `key_levels_align=daily`），返回与 bars 等长的六档序列；熔断逐 K 取 as-of 档位（无未来函数）。
- **Chat API**：`use_key_level_episode_stop`；`key_break_buy_ref` / `key_break_sell_ref`（默认 `support_low` / `resist_high`，可换任意 support_* / resist_*）。
- **Eval**：`workflow_signal_diagnose_sar_tencent` 默认开启。

## 与 playbook 区别

- 本 workflow **串行固定步骤**，不经过 ReAct。
- **不调用** `run_strategy_backtest`。
