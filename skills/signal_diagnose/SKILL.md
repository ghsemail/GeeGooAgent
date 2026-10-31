---
name: signal_diagnose
description: 信号诊断：读策略 → probe → Episode 评价 → diagnose → LLM 总结 → 写入知识库（不回测）。Chat 用 skill_view 加载后按步调普通工具。
trigger_modes: chat
---

# 信号诊断（Chat · skill + 工具）

## 何时使用

- 「**诊断 SAR 策略 · 腾讯**」「信号诊断 Macd4H · 00700」
- Monday 面板拼好的诊断文案（含策略摘要与标的）

## Chat 执行顺序（不要用 run_chat_workflow_turn）

**推荐首步（非硬性）**：从注入的 **Skills catalog** 选中 `signal_diagnose` 后，先 **skill_view**(`skill=signal_diagnose`) 加载本 SKILL.md，再按下列步骤调用普通工具（Eval Live 不强制本步）。

1. **解析意图**：从用户话里提取 `strategy_query`（策略/catalog 名）与 `stock_query`（公司/代码）。缺一项 → **clarify**。
2. **读策略**：用户只给指标/模糊名 → 先 **search_signals** 列候选，2+ 命中 → **clarify**；已有明确 catalog 名时按序解析（与 Workflow Runner 一致）：**get_signal_combinations** → **get_index_signals** → **get_custom_signal_for_skill** → **get_custom_strategy_definitions**，得到 `buy_signal`、`sell_signal`；catalog 返回的 **frequency 只是候选**，勿直接当作最终 probe 周期。
3. **确认 frequency**（在 **run_probe** 之前必须完成）：
   - ① 用户**当句**已写明周期（60m / 15m / 5m / daily 等）→ 直接采用；
   - ② **Monday 面板 / sticky**（`workflow_options.signal_diagnose`、InboundPanelOps 或会话 sticky_panel_ops）已带 `frequency` → **视为已确认**，直接采用；回复里可一句对齐（例：「按面板 60m 诊断」）；**禁止**再 clarify「请选择 K 线周期」；
   - ③ 否则若会话记忆或用户偏好里有常用周期 → 采用，并在 probe 前用一句话告知（例：「按你常用的 60m 诊断」）；
   - ④ 否则 **必须 clarify**，选项建议 `60m` / `daily`（必要时加 15m / 5m）；**禁止**未确认就按 catalog 默认或 daily 调用 probe / diagnose。
   - 服务端会在 probe / diagnose 前自动注入面板 frequency / 买卖点；catalog 返回的 frequency 仍只是候选，不能代替上述确认。
4. **resolve_symbol**：**search_code** → 标准 `code`。
5. **run_probe**：**probe_bot_signal_series**（使用**已确认的 frequency**；`include_key_levels` / `key_levels_mode=series` 默认；`months_back` 按诊断窗，通常 3，或面板指定值）。
6. **（可选）fetch_key_levels**：probe 未带回关键价位时 **get_key_levels**。
7. **evaluate_accuracy**：在 probe 结果上按 SKILL 内 Episode 规则理解 Strict/Path（展示用；Go 工具已算好字段）。
8. **build_detail**：**diagnose_bot_signal_series**（同一 code、**已确认的 frequency**、buy_signal、months_back）。
9. **diag_compose**：用主模型根据 probe + diagnose + price_analysis 写综合判断 Markdown（勿编造 probe 未返回的数据）。
10. **diag_save_kb**：**save_strategy_knowledge** → 信号诊断目录；记录返回的 knowledge id。
11. **summarize**：向用户输出 Markdown 诊断报告（含知识库 id）。
12. **UI 卡片**：probe/diagnose 有结果后调用 **emit_workflow_card**，`card.skill=signal_diagnose`，`status=completed`，`phase=done`，`card_stage=eval_done`（或与 runner 一致的阶段）。只填**摘要字段**（`stock_code`、`strategy_query`、`diagnose_detail`、probe 命中、`probe_bar_count` 等）；**不要**在 card 里粘贴 `chart_probe.bars` 或完整 `signal_eval`。服务端会在存储完整 K 线 + Episode 后自动写入 **`diagnose_ui_key`**；Monday 用该 key 调 Agent `GET /v1/signal-diagnose/ui-payload` 拉取重数据。

## 工具参数要点

| 步骤 | 工具 |
| --- | --- |
| 标的 | search_code |
| 买卖点序列 | probe_bot_signal_series |
| 零信号/阈值解释 | diagnose_bot_signal_series |
| 持久化 | save_strategy_knowledge |
| Monday 卡 | emit_workflow_card |

## K 线关键价位参考线

- probe 传 `key_levels_mode=series`（默认 `key_levels_align=daily`），六档与 bars 等长；策略开发页 K 线展示买段 `support_high` / 卖段 `resist_low` 阶梯线。

## 价格分析 Episode 门控（默认关 / 软标注）

- **Chat API**：`use_price_analysis_episode_filter` 默认 **false**；false 时仍逐条计算四周期 `price_analysis` 与 `would_intercept` / `ai_intercept_conclusion`，**不**掩码剔除触发。
- **workflow_card / 诊断 Markdown**：`price_analysis`（5m/60m/日线/周线表 + 支撑阻力 + 综合句）；`price_episode_filter_stats.buy_triggers` / `sell_triggers` 列出全部买/卖触发。

## 与回测区别

- 本 skill **不调用** `run_strategy_backtest`。
- 用户只要 PnL 回测 → **strategy-backtest-run** + `run_strategy_backtest`。
