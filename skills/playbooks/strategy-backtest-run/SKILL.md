---
name: strategy-backtest-run
description: 策略回测、run_strategy_backtest、多标的回测、多策略对比、不同止盈止损对比、同样参数再跑。用户要「跑回测」「比一下这几只」「哪个策略赚得多」「1个月和3个月哪个好」时触发。
skip_retrieval_gate: true
---

# 策略回测 · 运行

路径 **A**：SmartTrade（`run_strategy_backtest`）· 路径 **B**：DCA/GRID（`loopback_strategy`）。

> 继承 **`strategy-backtest`**：参数 ①→⑤、**§三维 A/B/C 全覆盖**。

## 适用 Toolset

`strategy` · `custom_signal` · `market`

---

## 路径 A · 单标的

```
判定维数 → 解析参数（③优先）→ run_strategy_backtest →（可选 get）
```

**③ 同样/再跑**：`list` → get → 复用 `config` + `trade_config` + `period`  
**④ 默认**：fund 100000 · base 100 · trade_config 见父 playbook（SL 默认 dynamic SAR）  
**Macd4H/共振**：默认 **`months_back: 3`**（勿手算 limit；见 **`strategy-signal-probe`**）

主 Tool：**`run_strategy_backtest`**（probe+模拟+落库；`user_id`/`source` 自动注入）  
仅验信号 → **`strategy-signal-probe`**

必填与 probe 相同 + 可选：`strategy_label` · `period` · `fund` · `trade_config` · `sell_signal`（默认同 buy）

---

## 三维 · 本 playbook 覆盖（A/B/C）

**共性**：每轮只改一个维度；**最多 4 组**；每组一次 `run_strategy_backtest`；汇总并列表。

### A · 同策略多标的

| 项 | 内容 |
|----|------|
| 固定 | rules · frequency · **months_back/period** · **同一** trade_config |
| 变化 | `code` |
| memory | `list(strategy_label)` 不设 code → 已有各码收益可直接报；缺则循环 run |
| 输出列 | code · profit_rate · drawdown · trade_count · log_id |

### B · 同标的多策略

| 项 | 内容 |
|----|------|
| 固定 | `code` |
| 变化 | 每策略 rules + frequency + **months_back** + **配套** trade_config（共振→macd_resonance_v1） |
| memory | `list(code)` → 按 strategy_label 分组 |
| 输出列 | strategy_label · frequency · profit_rate · drawdown · log_id |

### C · 同标的同策略多配置

| 项 | 内容 |
|----|------|
| 固定 | `code` + buy/sell 链 |
| 变化 | **`period`** / **`months_back`** / `trade_config` / `fund` 等（每次只改一项或一组） |
| memory | `list(code, strategy_label)` — 列表含 **period**+profit_rate，常够粗比；细比 TP/SL → get 2～4 条 |
| 新跑 | 循环 run，仅改 config 块；记录 **配置摘要** + log_id |
| 输出列 | 配置摘要 · profit_rate · drawdown · trade_count · log_id |

**配置摘要模板**：`{period} · TP{fix_tp}% · SL {sl_mode}/{index} · fund {fund}`

**禁止**：仅用 probe 完成情形 C 的盈亏结论。

---

## 路径 B · DCA/GRID

父 playbook §DCA/GRID · memory 来自 `generate_*` · 非三维 A/B/C 的 SmartTrade 路径。

---

## 硬规则 · 反模式

- 禁止跳过 run 声称回测完成；禁止「同样参数」却不用 ③  
- 禁止 A 场景对每码用不同 trade_config（除非用户明确要求）  
- 禁止 B 场景强行统一 frequency（除非用户要求）

## 输出

单次：log_id + 收益摘要 + 参数来源 · **A/B/C 并列表** + 结论

免责声明：仅供参考，非投资建议。
