---
name: strategy-backtest-run
description: 策略回测运行、probe 后验证、loopback、高级策略 SmartTrade 回测、止盈止损、动态止损。用户要「跑回测」「验证策略收益」「最新回测」时触发。
---

# 策略回测 · 运行

覆盖两类回测路径，与 `trading_operation` **策略回测**页对齐。

## 适用 Toolset

`strategy` · `custom_signal` · `market`

## 路径 A：高级策略 / SmartTrade 式回测（trading_operation 新版）

新版回测在客户端模拟止盈止损后写入 Mongo；Agent **信号 + 读结果** 流程如下：

### 1. 确认参数

向用户确认或从上下文读取：

- `code`、`frequency`、`months_back` / `limit`
- `buy_signal` / `sell_signal` 规则
- 资金、仓位模式、止盈止损（固定 / 动态 / 跟踪）

### 2. 信号探测（必做）

`probe_bot_signal_series` — 与 UI「运行回测」第一步相同。

若动态 TP/SL：`get_indicator_series`（`role=sl` 或 `tp`，`index` 与 sl_tp 配置一致）

### 3. 完整 PnL 模拟

- **Agent 侧**：`probe` 只能验证信号，**不能**在服务端算 SmartTrade 盈亏
- **用户在 trading_operation 回测页点击运行**后，记录会写入 Mongo
- Agent 用 **`list_strategy_backtest_logs`**（`code` + 最新 `created_at`）→ **`get_strategy_backtest_log`** 读取完整结果

向用户说明：若需带止盈止损的精确盈亏，请在回测页运行一次，再由 Agent 读 `log_id`。

## 路径 B：DCA / GRID 服务端回测（loopback）

适合 Bot 方案验证，与旧版 `loopback_strategy` 一致：

1. `search_code`
2. `get_signal_combinations` 或 `get_index_signals` → 选 `signal_id`
3. `generate_dca_strategy` 或 `generate_grid_strategy`（告知等待时间）
4. `loopback_strategy`：
   - **grid**：`type=grid`，`grid_param` = generate 的 `param`，`frequency=5m`
   - **dca**：`type=dca`，`signal` = `signal.buy_signal`，`sl_tp` 由 `dynamicParam`/`fixedParam` 组装，`frequency=60m`

缺 `fund` / `months_back` 时先问用户。

## 硬规则

- **禁止**把 `probe` 的买卖次数直接说成收益率
- **禁止**无 `grid_param` / `signal`+`sl_tp` 裸调 `loopback_strategy`
- 动态/跟踪止损出场可能**盈利**，动作仍可能标「止损」— 以 `realized_pnl` 为准

## 反模式

- 跳过 probe 直接声称「回测完成」
- 混淆 `loopback_strategy`（DCA/Grid）与 trading_operation 高级回测

## 输出

- **probe 阶段**：信号统计 + 是否值得继续
- **loopback**：`finalValue`、`profit_rate`、`drawdown`、`annualized_return`
- **读历史**：`result` 摘要 + 关键成交（止盈/止损/买入）+ `log_id`

免责声明：仅供参考，非投资建议。
