---
name: strategy-backtest-history
description: 回测历史、list/get log、多标的收益对比、多策略收益对比、同策略不同配置对比、log_id。用户要「历史回测」「上次结果」「这几条哪个赚得多」时触发。
skip_retrieval_gate: true
---

# 策略回测 · 历史与结果查看

Mongo `strategy_backtest_log`（UI + **`run_strategy_backtest`**）· **`user_id` 隔离**。

> 继承 **`strategy-backtest`**：参数 ①→⑤、**§三维** 读历史分工。

## 适用 Toolset

`strategy` · `market`

---

## 单条流程

`list_strategy_backtest_logs`（limit=20）→ 必要时 clarify → `get_strategy_backtest_log` → 呈现

**list 摘要字段**：`log_id` · `code` · `strategy_label` · `period` · `frequency` · `created_at` · `trade_count` · `result.profit_rate` · `result.drawdown`

- 列表够用 **勿逐条 get**  
- 2+ 条要详情未指定 → clarify（date + profit_rate + period）

**get 详情**：`result` · `trades` · `config`（供 sibling 复跑）· `trade_config`（情形 C 细比）

---

## 三维 · 本 playbook 读历史分工

| 情形 | list 筛选 | 列表能否粗对比 | 何时 get |
|------|-----------|----------------|----------|
| **A 同策略多标的** | `strategy_label=…` **不设 code** | ✅ 按 **code** 列 profit_rate | 用户要某码成交细节 |
| **B 同标的多策略** | `code=…` **不设 strategy_label** | ✅ 按 **strategy_label** 列 | 用户要某策略 trades |
| **C 同码同策略多配置** | `code` + `strategy_label` | ✅ 按 **period** + profit_rate | 比 TP/SL 细节 · 成交差异 |

### A · 示例

「Macd4H 我之前测过哪些票、谁赚得多」→ `list(strategy_label=4小时…)` → 表：code | period | profit_rate | log_id

### B · 示例

「腾讯我跑过哪些策略」→ `list(code=00700.HK)` → 表：strategy_label | profit_rate | log_id

### C · 示例

「腾讯 Macd4H 1 月和 3 月哪次好」→ `list(code, strategy_label)` → 表：period | created_at | profit_rate | drawdown  
若比「TP5% vs 7%」→ list 无 trade_config → **get 选中的 2 条** 读 `trade_config`

### 衔接

- 再跑 / probe → **`strategy-backtest-run`** / **`strategy-signal-probe`**，用 get 到的 **config**（③）  
- 新 config 对比 → 路由 **backtest-run** 情形 C

---

## 硬规则 · 反模式

- 无 log_id 先 list · 不提供 delete · 禁止 dump chart_data  
- 禁止 list 每条都 get · 禁止 profit_rate 当实时建议

## 输出

列表表 · 单次详情+关键成交 · **A/B/C 对比并列表**（注明 log_id）
