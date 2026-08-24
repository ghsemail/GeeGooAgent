---
name: strategy-signal-probe
description: 策略开发信号测试、probe_bot_signal、probe_bot_signal_series、买卖信号验证、K 线信号序列、定制策略 Macd4HRhythm。用户要「测信号」「有没有买卖点」「策略开发测试」时触发。
skip_retrieval_gate: true
---

# 策略开发 · 信号测试

与 `trading_operation` **策略开发**页「信号测试」等价（GeeGooSignal `probeBotSignalSeries`）。

> **继承父 playbook `strategy-backtest`**：参数解析优先级 ①→⑤、买卖规则来源、Gateway `clarify` 规则。本节仅写 **probe 专有** 流程与默认值。

## 适用 Toolset

`strategy` · `custom_signal` · `market`（`search_code`）

---

## 流程

```
确认 code → 组装 buy/sell 规则 → 定 frequency + limit → probe_bot_signal_series → 解读
```

### 1. 确认标的

`search_code` → `code`、`name`、`lot_size`。crypto 用 `BTCUSDT` 等形式。

### 2. 组装规则（继承父 playbook 来源表）

- **单指标 / 组合 / 定制**：按父 playbook「买卖规则来源」取链  
- **用户说「上次回测那套 / 同样配置」**：先父 playbook **③** `list` → `get`，从 `run.config.buy_rules` / `sell_rules` 或 `chart_data.probe` 取规则；**勿**直接用 registry 覆盖  
- **定制策略 param**：默认 registry **`defaults`**；仅用户指定「我保存的那条」才读 `get_custom_signal_for_skill`

**probe 专有 · `sell_signal`**：单指标 / 定制策略与 UI 一致——未单独指定卖出时，**`sell_signal` = `buy_signal`**（同一 index 产出 ±1；仅 buy 会看不到卖出）。

### 3. frequency 与 limit（对齐 UI 策略开发页）

**frequency**：须在策略 `supported_frequencies` 内。

| 策略 | frequency |
|------|-----------|
| Macd4HRhythm | **`60m` 唯一** |
| MACDResonance | **`15m`**（或 `5m`） |
| 单指标 / 组合 | catalog 项 |

**limit（优先于 `months_back`）**：UI 只传 `limit`。服务端 `months_back` 推算按股票 **7 根/天**（60m），crypto 须自行算 limit。

| 场景 | limit 建议 |
|------|------------|
| 一般 60m 股票 · 约 1 月 | **154**（22×7） |
| **Macd4HRhythm** | **≥450**；默认 **3 月 ≈462**（89 均线预热） |
| **MACDResonance** | 建议 **3 月**（信号稀疏） |
| Crypto 60m | `min(24×天数, **800**)` |
| 快速扫一眼（非 Macd4H） | 100～300 |

`months_back` 工具 schema 默认 3；**对齐 UI 时请显式传 `limit`**，勿单靠 `months_back`。

#### 定制策略 defaults（registry）

**MACDResonance**（`15m`/`5m`）：fastPeriod 12 / slowPeriod 26 / signalPeriod 9 · zeroAxisRatio 0.002 · crossTouchRatio 0.002 · breakoutLookback 5 · enablePseudoCross true · highTfLoadLimit 400

**Macd4HRhythm**（`60m`）：fastPeriod 5 / slowPeriod 13 / signalPeriod 9 · rhythmPeriod 89 · zone1Ratio 0.0015 · zone3Ratio 0.0045 · minRoundBars 5 · enableExtremeReversal true

### 4. 调用

| 场景 | Tool |
|------|------|
| 整段 K 线触发序列 | **`probe_bot_signal_series`** |
| 单 bar（默认最后一根） | `probe_bot_signal`（可选 `at`） |

**最小可用示例**（Macd4HRhythm · 对齐 UI）：

```json
{
  "code": "00700.HK",
  "frequency": "60m",
  "limit": 462,
  "buy_signal": [{
    "index": "Macd4HRhythm",
    "type": "signal",
    "param": {
      "fastPeriod": 5, "slowPeriod": 13, "signalPeriod": 9,
      "rhythmPeriod": 89, "zone1Ratio": 0.0015, "zone3Ratio": 0.0045
    }
  }],
  "sell_signal": [{
    "index": "Macd4HRhythm",
    "type": "signal",
    "param": {
      "fastPeriod": 5, "slowPeriod": 13, "signalPeriod": 9,
      "rhythmPeriod": 89, "zone1Ratio": 0.0015, "zone3Ratio": 0.0045
    }
  }]
}
```

### 5. 解读

- `buy_merged`：`1` = 买；`sell_merged`：`-1` = 卖  
- **零信号**：查 frequency、limit 是否过低、param 是否过严、`sell_signal` 是否缺失  
- 汇报：**买卖次数 + 最近 3 次触发时间**；`invalid` / `error` 单独列出  

---

## 配置推断（在父 playbook ①～④ 之后）

| 用户说法 | 推断 |
|----------|------|
| 「按 UI / 默认」+ Macd4HRhythm | 60m · limit≥450（3 月）· registry defaults · sell 镜像 |
| 「按 UI / 默认」+ MACDResonance | 15m · 3 月 limit · defaults |
| 「上次 / 同样 / 那次回测配置」 | **③** get log → config 或 chart_data.probe |
| 「MACD 共振」 | MACDResonance · 15m |
| 「4 小时节奏」 | Macd4HRhythm · 60m |
| 已给 code + 策略名 | 直接 probe，勿逐项问周期 |

---

## 硬规则

- 必填：`code` + `frequency` + 至少一条 `buy_signal`  
- 定制 `index` 须在 `get_custom_strategy_definitions` 注册表中  
- 最多 **800** bar；禁止 dump 全量 `bars` / `signal_series`  
- 完整 PnL → 路由 **`strategy-backtest-run`** 或 **`strategy-backtest-history`**

## 反模式

- 未 `search_code` 硬编码 code  
- Macd4HRhythm 用 `months_back=1` 或不传 sell（易 0 信号 / 无卖出）  
- 把 probe 当最终收益率  

## 输出

表格：标的、周期、limit/回溯、买/卖触发次数、最近 3 次信号时间、参数来源（明文/历史/默认）。
