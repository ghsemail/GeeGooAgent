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

## 价格分析 Episode 门控（默认关 / 软标注）

- **Chat API**：`use_price_analysis_episode_filter` 默认 **false**；false 时仍逐条计算四周期 `price_analysis` 与 `would_intercept` / `ai_intercept_conclusion`（本会拦截），**不**掩码剔除触发；true 时恢复硬掩码（与旧行为一致）。
- **门控综合分模式（实验）**：`price_analysis_gate_score_mode` 默认 **`full`**（门控分=展示 `price_score`，含进场周期）；设为 **`htf_only`** 时展示分不变，买点 `would_intercept` 仅用更大周期（如 60m 信号仅周+日，权重 0.45/0.35 归一）；触发记录带 `gate_price_score`。
- **workflow_card / 诊断 Markdown**：`price_analysis`（5m/60m/日线/周线表 + 支撑阻力 + 综合句）；`price_episode_filter_stats.buy_triggers` / `sell_triggers` 列出全部买/卖触发（`would_intercept=true` 为按规则本会硬拦），与 `stripped_*_triggers` 并存。
- **取 K 编排（Agent）**：打开 Episode 过滤时，四周期各 **一次** `probe_bot_signal_series`（`months_back` 按诊断窗扩展，`as_of`=图表最晚 bar）；每条触发在本地按「前一交易日 16:00」锚点截断后再分析。**依赖 GeeGooSignal `as_of`/`end`**；未合入前 probe 仍可能只返回「从现在往前 ≤800 根」导致早期 5m「锚点前无 K」。
- **单点四周期表**：代码入口 `strategydev.BuildRunPriceAnalysisAsOf(ctx, run, code, asOf, months, …)`；probe 传 `as_of` + `months_back`（默认 3），不传 `as_of` 时 `BuildRunPriceAnalysis` 保持旧「最新窗」行为。

## 与 playbook 区别

- 本 workflow **串行固定步骤**，不经过 ReAct。
- **不调用** `run_strategy_backtest`。
