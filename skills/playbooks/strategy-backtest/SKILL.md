---
name: strategy-backtest
description: 策略回测、网格策略、DCA 定投方案、generate_grid、loopback、信号组合、单指标信号、MACD、SAR、参数建议。用户说「回测」「网格怎么设」「DCA 方案」「哪个信号好」时触发。
---

# 策略生成与回测

> **细分 playbook**：信号测试 → `strategy-signal-probe`；运行回测 → `strategy-backtest-run`；读历史 → `strategy-backtest-history`

## 适用 Toolset

`strategy` · `market`（`search_code`）· `custom_signal`（若用定制信号）

## 标准流程

### 1. 确认标的

`search_code` → 得到 `code`、`name`

### 2. 选信号类型（用户未指定时 clarify）

| 路径 | Tool | 适用 |
|------|------|------|
| 单指标 | `get_index_signals` | SAR、MACD、BBAND 等 |
| 组合共振 | `get_signal_combinations` | 多指标 buy/sell 链 |
| 高级/定制 | `get_custom_signal_for_skill` | Macd4HRhythm 等 |

展示 **brief/info**，让用户选定 **`signal_id`**（DCA）或理解 grid 不需要 signal_id。

### 3. 信号测试（策略开发）

`probe_bot_signal_series` — 见 playbook **strategy-signal-probe**

### 4. 生成方案（可选）

| 类型 | Tool | 耗时提示 |
|------|------|----------|
| 网格 | `generate_grid_strategy`（`code`, `name`） | cn ~40s |
| DCA | `generate_dca_strategy`（`code`, `name`, `signal_id`） | cn ~2min |

### 5. 回测

| 场景 | Tool |
|------|------|
| DCA/GRID 服务端回测 | `loopback_strategy`（须先有 generate 参数） |
| 高级策略完整 PnL | 用户在 trading_operation 运行 → `list_strategy_backtest_logs` / `get_strategy_backtest_log` |

见 playbook **strategy-backtest-run**、**strategy-backtest-history**。

`loopback_strategy` 需要完整入参：

- **grid**：`type=grid`，`grid_param` 来自 `generate_grid_strategy` 的 `param`
- **dca**：`type=dca`，`signal` 来自 `generate_dca_strategy.signal.buy_signal`，`sl_tp` 按 `comparison` 选 `dynamicParam` 或 `fixedParam`

缺 `fund` / `months_back` 时先问用户。

### 6. 定制信号（可选）

若用户要用 MACDResonance 等：`get_custom_strategy_definitions` → `get_custom_signal_for_skill` → `probe_bot_signal_series` 验证。

## 硬规则

- **禁止**无 `grid_param` / `signal`+`sl_tp` 直接 `loopback_strategy`
- **禁止**把 `probe` 结果当最终收益率
- 生成与回测是**两步**；向用户说明等待时间
- 回测结果注明区间、资金、信号来源

## 反模式

- 跳过信号选择直接 generate_dca
- 把 generate 输出当最终交易建议而不解释风险

## 输出

表格或列表：信号名、关键参数、回测收益/回撤（若有）；明确「仅供参考，非投资建议」。
