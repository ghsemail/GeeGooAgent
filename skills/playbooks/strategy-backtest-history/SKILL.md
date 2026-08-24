---
name: strategy-backtest-history
description: 策略回测历史、list_strategy_backtest_logs、get_strategy_backtest_log、回测结果查看、盈亏、时间线、log_id。用户要「历史回测」「上次回测结果」「打开某次回测」时触发。
skip_retrieval_gate: true
---

# 策略回测 · 历史与结果查看

读取 `trading_operation` 策略回测页与 Agent **`run_strategy_backtest`** 写入的 Mongo 记录（`strategy_backtest_log`）。**按当前用户 `user_id` 隔离**；未带 `user_id` 的旧记录不会出现在列表中。

> **继承父 playbook `strategy-backtest`**：参数解析 ①→⑤、list/get memory、clarify 选 log。  
> **本 playbook 重点**：读 **`result` / trades** 汇报；`config` 供 sibling **`strategy-backtest-run`** / **`strategy-signal-probe`** 复用参数。

## 适用 Toolset

`strategy` · `market`（按 code 筛选时可 `search_code`）

---

## 流程

```
list（筛选）→ 必要时 clarify 选 log → get 读详情 → 向用户呈现
```

### 1. 列历史

`list_strategy_backtest_logs` — **优先直接调用**，默认 `limit=20`；勿 delegate。**`user_id` 由运行时自动注入**，只返回当前用户记录。

| 参数 | 用途 |
|------|------|
| `code` | 如 `1810.HK` |
| `strategy_label` | 策略显示名 |
| `date_from` / `date_to` | `YYYY-MM-DD` |
| `limit` / `skip` | 分页 |

摘要：`log_id`、`code`、`strategy_label`、`created_at`、`trade_count`、`result.profit`、`result.profit_rate`、`has_chart_data`

- 回复列表：**表格即可**，勿为每条再 `get`  
- **2+ 条且用户要详情未指定哪条** → 父 playbook **⑤ clarify** 选 `log_id`  
- 用户说「上次」且仅 **1 条** 匹配 → 直接 `get`

### 2. 读详情

`get_strategy_backtest_log`（`log_id`）

| 区块 | 本 playbook |  sibling 复跑 |
|------|-------------|----------------|
| `run.result` | **汇报盈亏** | — |
| `trades` | 成交时间线 | — |
| `run.config` · `trade_config` · `frequency` · `period` | 概览可简述 | **probe / 再跑入参** |
| `run.chart_data.probe` | 仅按需摘录 | probe 快照 |

字段说明：

- `run.result`：`final_value`、`profit`、`profit_rate`、`drawdown`、`annualized_return`、`closed_trades`  
- `run.config`：`buy_rules`、`sell_rules`（及嵌套 `trade_config` 若存在）  
- `trades`：`action`、`action_label`、`trade_price`、`realized_pnl`  

### 3. 向用户呈现

- **概览**：标的、策略、区间、收益率、回撤、笔数、`log_id`  
- **重点成交**：首买、止盈、止损（含盈利止损出场）  
- **无 `has_chart_data`**：仅数字，勿编造 K 线  

### 4. 与「再跑 / 同样参数」的衔接

用户读完历史后要 **再跑或 probe** → 路由 **`strategy-backtest-run`** 或 **`strategy-signal-probe`**，用已得 `log_id` 的 **config**（父 playbook ③），勿重新猜默认。

---

## 硬规则

- 无 `log_id` → 先 `list`  
- Agent **不提供** delete  
- 大字段只摘录相关 bar，禁止全文 dump  

## 反模式

- 列表 `profit_rate` 当实时建议  
- 混淆 Bot `loopback` 与本策略回测历史  
- 用户只要列表却对每条 `get`  

## 输出

Markdown 表格：时间、标的、策略、收益率、回撤、笔数；详情附 3～5 笔关键成交与 `log_id`。
