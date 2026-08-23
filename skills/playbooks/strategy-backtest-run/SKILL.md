---
name: strategy-backtest-run
description: 策略回测运行、probe 后验证、loopback、高级策略 SmartTrade 回测、止盈止损、动态止损。用户要「跑回测」「验证策略收益」「最新回测」时触发。
skip_retrieval_gate: true
---

# 策略回测 · 运行

覆盖两类回测路径，与 `trading_operation` **策略回测**页对齐。

## 适用 Toolset

`strategy` · `custom_signal` · `market`

## 路径 A：高级策略 / SmartTrade 式回测（trading_operation 新版）

新版回测在客户端模拟止盈止损后写入 Mongo；Agent **信号 + 读结果** 流程如下：

### 1. 确认参数（用户未指定时用 UI 默认值）

向用户确认或从上下文读取；**能推断则直接采用默认，勿逐项追问**。

#### 行情与回溯

| 字段 | 默认值 | 可选值 | 说明 |
|------|--------|--------|------|
| `code` | — | 须 `search_code` | 如 `00700.HK` |
| `frequency` | **`60m`** | `5m` / `15m` / `60m` / `daily` | 定制策略须在其 `supported_frequencies` 内 |
| `period`（UI 回溯） | **`1m`** | 5m 周期：`2w`/`1m`/`2m`；其他：`1m`/`2m`/`3m` | 映射为 probe 的 `months_back` |
| `months_back` | **1**（来自 period） | 1～12 | probe / 回测 API 用整数月 |
| `limit` | 按 months 推算 | 30～800 | 策略开发可只传 limit |
| `fund` | **100000** | 100000 / 200000 / 500000 或自定义 | 初始资金（随市场币种） |
| `base_order_size` | **100** | 100 / 200 / 500 或自定义 | 每次买入股数；crypto 为 USDT 额 |

#### 买卖规则来源（Monday catalog-api）

| 策略类型 | Tool | 取参方式 |
|----------|------|----------|
| 单指标 | `get_index_signals` | 列表选 `signal_id`；probe 用该项的 `index`+`param` 模板 |
| 组合 | `get_signal_combinations` | 列表选 `signal_id`；probe 用该项的 **`buy_signal` / `sell_signal` 整条链** |
| 高级/定制 | `get_custom_signal_for_skill` + `get_custom_strategy_definitions` | `buy_signal` 项：`index`=注册表 `strategy_key`，`param` 用 `defaults` 或用户覆盖 |

**多匹配时必须 clarify**：组合/单指标列表有 2+ 项都符合用户描述时，先 `clarify`（最多 4 项）再回测，勿默认第一个。

**定制策略注册表（`get_custom_strategy_definitions`）**

| strategy_key | 默认 frequency | supported_frequencies | 默认 param（节选） |
|--------------|----------------|----------------------|-------------------|
| `MACDResonance` | 15m | `15m`, `5m` | fastPeriod=12, slowPeriod=26, signalPeriod=9, zeroAxisRatio=0.002, breakoutLookback=5 |
| `Macd4HRhythm` | 60m | `60m` | fastPeriod=5, slowPeriod=13, signalPeriod=9, rhythmPeriod=89, zone1Ratio=0.0015 |

`custom.type` 枚举：`signal` | `flag`（须在该策略 `supported_types` 内）。

#### 交易参数 `trade_config`（与 UI「交易参数」三 Tab 对齐）

**执行模式 `execution_profile`**

| 值 | 标签 | 何时用 |
|----|------|--------|
| `generic_smarttrade` | 标准止盈止损 | **默认**；通用指标/组合 |
| `macd_resonance_v1` | 策略配套 | 仅当 **全部** buy 规则 `index=MACDRESONANCE` 时 UI 自动切换 |

**止盈 `tp`（默认）**

| 字段 | 默认 | 枚举/范围 |
|------|------|-----------|
| `tp_switch` | true | |
| `tp_mode` | **`fix`** | `fix` / `dynamic` |
| `fix_tp` | **5%** | 预设 3 / 5 / 7 |
| `tp_dynamic_index` | SAR | SAR / BBAND（`tp_mode=dynamic` 时） |
| `tp_dynamic_factor` | 1.0 | |
| `profit_trailing` | true | 盈利回撤跟踪 |
| `profit_trailing_deviation` | 1% | |

**止损 `sl`（默认）**

| 字段 | 默认 | 枚举/范围 |
|------|------|-----------|
| `sl_switch` | true | |
| `sl_mode` | **`dynamic`** | `fix` / `dynamic` |
| `sl_dynamic_index` | **SAR** | SAR / BBAND（`sl_mode=dynamic` 时） |
| `fix_sl` | 3% | 仅 `sl_mode=fix` 时生效；预设 3 / 5 / 7 |
| `stop_loss_trailing` | false | |
| `stop_loss_trailing_deviation` | 1% | |

**出场**：`use_signal_sell` 默认 **true**（信号卖出 + 止盈止损并存）。

**仓位 `position`（默认）**

| 字段 | 默认 | 枚举 |
|------|------|------|
| `sizing_mode` | **`fixed`** | `fixed` 固定股数 / `riskBased` 按资金比例 |
| `base_order_size` | 100 | |
| `position_mode` | **`addOn`** | `single` 单次建仓 / `addOn` 允许加仓 |
| `per_trade_risk_pct` | 2% | `riskBased` 时用 |
| `cap_by_fixed_order_size` | true | |

**风控 `risk`（默认）**

| 字段 | 默认 |
|------|------|
| `per_trade_enabled` | false |
| `monthly_halt_enabled` | **true** |
| `monthly_loss_limit_pct` | **6%** |

**MACD 配套执行 `macd_exec`**（仅 `macd_resonance_v1`）：breakoutLookback=5, atrPeriod=14, perTradeRiskPct=2%, takeProfitHalfR=2, breakevenR=3；5m/15m 小周期需大周期 `60m` MACD 反向出场。

用户只说「默认回测」「按常规」→ 用上表默认 + 选定信号的 buy/sell 链，**不要**展开问止盈止损。

### 2. 信号探测（必做）

`probe_bot_signal_series` — 与 UI「运行回测」第一步相同。

入参示例：

```json
{
  "code": "00700.HK",
  "frequency": "60m",
  "months_back": 1,
  "buy_signal": [{"index": "Macd4HRhythm", "type": "signal", "param": {"fastPeriod": "5", "slowPeriod": "13", "signalPeriod": "9"}}],
  "sell_signal": []
}
```

若 `tp_mode`/`sl_mode`=`dynamic`（**止损默认 dynamic**）：`get_indicator_series`（`role=sl` 或 `tp`，`index` 与配置一致，止损默认 `SAR`）。

### 3. 完整 PnL 模拟

- **Agent 侧**：`probe` 只能验证信号，**不能**在服务端算 SmartTrade 盈亏
- **用户在 trading_operation 回测页点击运行**后，记录写入 Mongo
- Agent 用 **`list_strategy_backtest_logs`** → **`get_strategy_backtest_log`** 读取结果

向用户说明：若需带止盈止损的精确盈亏，请在回测页运行一次，再由 Agent 读 `log_id`。

## 路径 B：DCA / GRID 服务端回测（loopback）

适合 Bot 方案验证，与旧版 `loopback_strategy` 一致：

1. `search_code`
2. `get_signal_combinations` 或 `get_index_signals` → 选 `signal_id`
3. `generate_dca_strategy` 或 `generate_grid_strategy`（告知等待时间）
4. `loopback_strategy`：
   - **grid**：`type=grid`，`grid_param` = generate 的 `param`，`frequency=5m`
   - **dca**：`type=dca`，`signal` = `signal.buy_signal`，`sl_tp` 由 `dynamicParam`/`fixedParam` 组装，`frequency=60m`

缺 `fund` / `months_back` 时用 **100000 / 1**。

## 硬规则

- **禁止**把 `probe` 的买卖次数直接说成收益率
- **禁止**无 `grid_param` / `signal`+`sl_tp` 裸调 `loopback_strategy`
- 动态/跟踪止损出场可能**盈利**，动作仍可能标「止损」— 以 `realized_pnl` 为准
- 列表类 catalog 返回已摘要；**probe 前**若需完整 `buy_signal` 链，用户选定 `signal_id` 后从组合项取用（勿重复拉全量 info）

## 反模式

- 跳过 probe 直接声称「回测完成」
- 混淆 `loopback_strategy`（DCA/Grid）与 trading_operation 高级回测
- 用户只问「有哪些组合策略」时 dump 全量 `info` 或完整规则 JSON

## 输出

- **probe 阶段**：信号统计 + 是否值得继续（买卖次数、最近 3 次触发时间）
- **loopback**：`finalValue`、`profit_rate`、`drawdown`、`annualized_return`
- **读历史**：`result` 摘要 + 关键成交 + `log_id`

免责声明：仅供参考，非投资建议。
