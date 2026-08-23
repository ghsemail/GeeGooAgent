---
name: strategy-backtest-history
description: 策略回测历史、list_strategy_backtest_logs、get_strategy_backtest_log、回测结果查看、盈亏、时间线、log_id。用户要「历史回测」「上次回测结果」「打开某次回测」时触发。
---

# 策略回测 · 历史与结果查看

读取 `trading_operation` 策略回测页保存的 Mongo 记录（`strategy_backtest_log`）。

## 适用 Toolset

`strategy` · `market`（按 code 筛选时可选 `search_code`）

## 标准流程

### 1. 列历史

`list_strategy_backtest_logs`

| 参数 | 用途 |
|------|------|
| `code` | 如 `1810.HK` |
| `strategy_label` | 策略显示名 |
| `date_from` / `date_to` | `YYYY-MM-DD` |
| `limit` / `skip` | 分页，默认 limit=100 |

摘要字段：`log_id`、`code`、`strategy_label`、`created_at`、`trade_count`、`result.profit`、`result.profit_rate`、`has_chart_data`

### 2. 读详情

`get_strategy_backtest_log`（`log_id`）

| 区块 | 内容 |
|------|------|
| `run.result` | `final_value`、`profit`、`profit_rate`、`drawdown`、`annualized_return`、`closed_trades` |
| `run.config` | `buy_rules`、`sell_rules`、`trade_config`（止盈止损） |
| `run.chart_data.probe` | K 线 + `buy_merged` / `sell_merged`（与 probe 同结构） |
| `run.chart_data.snapshots` | 每 bar 持仓、成本、浮盈 |
| `trades` | 时间线： `action`、`action_label`、`trade_price`、`realized_pnl` |

### 3. 向用户呈现

- **概览**：标的、策略、区间、收益率、最大回撤、成交笔数
- **重点成交**：首次买入、止盈、止损（含「止损(锁定盈利)」类盈利出场）
- **无 `has_chart_data`**：仅数字摘要，勿编造 K 线

## 硬规则

- 无 `log_id` 时先 `list`，取最新一条或让用户指定
- 删除记录用 UI；Agent **不提供** delete tool（避免误删）
- 大字段（`chart_data`）只提取与用户问题相关的 bar，勿全文输出

## 反模式

- 把列表里 `profit_rate` 当实时建议
- 混淆 Bot `loopback` 日志与本策略回测历史

## 输出

Markdown 表格：时间、标的、策略、收益率、回撤、笔数；详情附 3～5 笔关键成交与 `log_id` 供 UI 对照。
