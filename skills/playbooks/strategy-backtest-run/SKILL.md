---
name: strategy-backtest-run
description: 策略回测运行、probe 后验证、loopback、高级策略 SmartTrade 回测、止盈止损、动态止损。用户要「跑回测」「验证策略收益」「同样参数再跑」「最新回测」时触发。
skip_retrieval_gate: true
---

# 策略回测 · 运行

覆盖 **路径 A**（高级 / SmartTrade）与 **路径 B**（DCA/GRID `loopback`）。与 UI **策略回测**页对齐。

> **继承父 playbook `strategy-backtest`**：参数解析 ①→⑤、memory、clarify、买卖规则来源。  
> **本 playbook 重点**：③ 历史 log **优先于** UI 默认表；完整 PnL 用 **`run_strategy_backtest`**。

## 适用 Toolset

`strategy` · `custom_signal` · `market`

---

## 流程（路径 A · 高级 / SmartTrade）

```
解析参数（①→⑤，③ 优先）→ run_strategy_backtest →（可选 get 详情）
```

仅验证信号、不要 PnL 时 → **`probe_bot_signal_series`**（见 **`strategy-signal-probe`**）。

### 1. 解析参数

**顺序**：父 playbook **① 明文 → ② 会话 → ③ list/get 回测 log → ④ 默认表 → ⑤ clarify**。

用户说 **「同样 / 再跑 / 上次 / 按那次回测」** 时：

1. `list_strategy_backtest_logs`（`code` / `strategy_label`，`limit=5`）  
2. 唯一 → `get_strategy_backtest_log(log_id)`  
3. 从 `run` 提取下表字段组装 **`run_strategy_backtest`** / probe 入参  

| run 字段 | 用途 |
|----------|------|
| `code` · `frequency` · `period` | 标的与周期 |
| `config.buy_rules` · `config.sell_rules` | 买卖链 |
| `trade_config` | 止盈止损 · 仓位 · 风控 · MACD 配套 |
| `fund` · `base_order_size` · `is_crypto` | 资金与下单规模 |
| `chart_data.probe` | 可选：还原 probe 范围 |

**多条历史** → `clarify` 选 `log_id`（带日期、收益率）。**无历史** → ④ 默认表。

#### ④ 默认表（仅无 ①②③ 时）

**行情与回溯**

| 字段 | 默认 | 说明 |
|------|------|------|
| `code` | — | `search_code` |
| `frequency` | **`60m`** | 须在 `supported_frequencies` 内 |
| `period` | **`1m`** | 5m：`2w`/`1m`/`2m`；其他：`1m`/`2m`/`3m` |
| `limit` | 见 **`strategy-signal-probe`** | Macd4H **≥450**；crypto 显式 limit |
| `fund` | **100000** | |
| `base_order_size` | **100** | crypto 为 USDT 额 |

**Macd4HRhythm / MACDResonance**：UI 默认回溯 **3 月**（非通用 1 月）——与 signal-probe playbook 一致。

**买卖规则**：父 playbook 来源表；定制 param 默认 registry **`defaults`**。

**`trade_config`（UI 默认 · 用户说「按常规」即用，勿逐项问）**

| 块 | 默认要点 |
|----|----------|
| `execution_profile` | `generic_smarttrade`；全 MACDResonance → `macd_resonance_v1` |
| `tp` | 开 · fix **5%** · 跟踪回撤 1% |
| `sl` | 开 · **dynamic SAR** · fix 3% 仅 fix 模式 |
| `use_signal_sell` | **true** |
| `position` | fixed · base 100 · addOn |
| `risk` | 月度熔断 **6%** 开 |

MACD 配套 `macd_exec`：breakoutLookback 5 · atrPeriod 14 · perTradeRiskPct 2% · takeProfitHalfR 2 · breakevenR 3。

### 2. 运行回测（路径 A 主路径）

**优先** **`run_strategy_backtest`**（服务端 probe + SmartTrade 模拟 + 写入 `strategy_backtest_log`），返回 `log_id`、`profit_rate`、`final_value`。

- 入参与 `probe_bot_signal_series` 相同，另可传 `strategy_label`、`fund`、`base_order_size`、`trade_config`、`period` 等  
- **`user_id` / `source=agent` 由运行时自动注入，勿手写**  
- `tp_mode`/`sl_mode=dynamic` 时服务端会拉 indicator 数据，**无需** Agent 单独调 `get_indicator_series`  
- **`sell_signal`**：单指标/定制策略默认同 `buy_signal`（见 signal-probe playbook）

示例（Macd4HRhythm · 含 sell 镜像）：

```json
{
  "code": "00700.HK",
  "frequency": "60m",
  "limit": 462,
  "buy_signal": [{"index": "Macd4HRhythm", "type": "signal", "param": {"fastPeriod": 5, "slowPeriod": 13, "signalPeriod": 9, "rhythmPeriod": 89}}],
  "sell_signal": [{"index": "Macd4HRhythm", "type": "signal", "param": {"fastPeriod": 5, "slowPeriod": 13, "signalPeriod": 9, "rhythmPeriod": 89}}]
}
```

回测完成后可用 **`get_strategy_backtest_log(log_id)`** 读详情；或 **`list_strategy_backtest_logs`** 列当前用户历史（详见 **`strategy-backtest-history`**）。

**备选**：用户在 `trading_operation` 回测页运行也会写入同库（`source=trading_operation`），Agent 用 list/get 读取即可。

**注意**：`probe_bot_signal_series` **仅**验证信号，**不能**单独声称已完成带止盈止损的盈亏回测。

---

## 路径 B：DCA / GRID · loopback

见 **父 playbook §DCA/GRID**（`generate_*` → `loopback_strategy`）。  
memory 来自 **generate 输出**，不是 `strategy_backtest_log`。缺 `fund`/`months_back` → **100000 / 1**。

---

## 硬规则

- **禁止**把 probe 买卖次数当收益率  
- **禁止**无 generate 参数裸调 `loopback_strategy`  
- **禁止**用户说「同样参数」却跳过 ③ 直接用 registry 默认  
- 动态止损出场可能盈利仍标「止损」— 以 `realized_pnl` 为准  

## 反模式

- 跳过 **`run_strategy_backtest`** / probe 直接声称「回测完成」  
- 混淆 loopback 与 SmartTrade 回测  
- 为列表问题 dump 全量 `info` / rules JSON  

## 输出

- **`run_strategy_backtest`**：`log_id`、`profit_rate`、`final_value`、`trade_count`（已落库）+ 参数来源（历史 log / 默认）  
- **probe 阶段**（仅探测时）：信号统计 + 最近 3 次触发  
- **loopback**：`finalValue`、`profit_rate`、`drawdown`、`annualized_return`  

免责声明：仅供参考，非投资建议。
