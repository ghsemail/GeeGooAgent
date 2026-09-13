---
name: strategy-signal-explain
description: 解释为什么没信号、零信号原因、阈值为何未触发、信号怎么算的。用户已测过信号或会话里已有 probe 参数后追问「为什么没信号」时触发。
skip_retrieval_gate: true
---

# 策略开发 · 信号诊断（为什么没信号）

与 **`strategy-signal-probe`** 配对：probe 负责「有没有触发」；本 playbook 负责「**为什么**没触发」。

## 适用 Toolset

`strategy` · `custom_signal` · `market`

---

## 何时触发

- 「为什么一个信号都没有 / 为什么没买入 / 怎么算的 / 阈值有没有碰到」
- 上一轮已 `probe_bot_signal_series` 或 Web 左栏已有相同 code+策略+频率

## 禁止

- **禁止**再次调用 `probe_bot_signal_series`（除非用户明确要求换参数重测）
- **禁止** `run_strategy_backtest` 代替解释
- 禁止无数据时编造 RSI/MACD 数值

---

## 流程

```
沿用会话 probe 参数 → diagnose_bot_signal_series → 读 summary + buy_rules → 中文解释
```

1. **参数来源（优先级）**
   - 会话里最近一次成功的 `probe_bot_signal_series` / `diagnose_bot_signal_series` / `run_strategy_backtest` 的 code、frequency、buy_signal、sell_signal、months_back
   - SessionTaskState / 用户刚确认的 clarify 选项
   - 仍缺 code 或策略 → `search_code` 或 `clarify`（仅补槽位）

2. **调用 `diagnose_bot_signal_series`**
   - 必填：`code`、`frequency`、`buy_signal`（与 probe 一致）
   - 默认：`months_back: 3`（与 probe 相同）
   - `side`：用户问买入→`buy`；问卖出→`sell`；都要→`both`

3. **解读响应**
   - `summary` + `buy_rules[].param`（参数必须复述给用户）
   - `buy_rules[].algorithm`（触发规则类型）
   - `stats.columns`（指标 min/max/last）
   - `last_bar` / `near_miss` / `sample_reasons`
   - `merged.buy_explain`（多规则 AND 时）

4. **输出结构**
   - 参数（含 period、阈值等）
   - 回溯区间与 K 线根数
   - 指标统计 + 最后 bar 判定
   - 一句结论（为何 0 触发或触发了几次）
   - 可选：若用户问「怎样才能出信号」，可建议调 months_back / frequency / 换策略类型（不自动重跑）

---

## 单指标（当前 SSOT）

每条 `buy_rules[]` 含完整 `param`；RSI/MACD/SAR/BBAND 等均走同一 diagnose 接口。

组合信号（多条 buy_rules）时说明 **AND 合并**：各规则单独 trigger_count vs `hits.buy`。

---

## 硬规则

- 必须先有 tool 结果再解释；禁止跳过 `diagnose_bot_signal_series` 声称「因为 RSI 太高」
- 回复引用 tool 中的数字，不要四舍五入到整数 unless 原值如此
