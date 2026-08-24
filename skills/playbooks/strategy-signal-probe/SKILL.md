---
name: strategy-signal-probe
description: 策略信号测试、probe、买卖点、多标的测信号、多策略比信号、Macd4HRhythm。用户要「测信号」「有没有买卖」「几只一起测信号」「哪个策略信号多」时触发。
skip_retrieval_gate: true
---

# 策略开发 · 信号测试

等价于 `trading_operation` 策略开发页 · Tool：`probe_bot_signal_series` / `probe_bot_signal`。

> 继承 **`strategy-backtest`**：参数 ①→⑤、规则来源、Gateway、**§三维场景**（本 playbook 仅 **A/B**）。

## 适用 Toolset

`strategy` · `custom_signal` · `market`

---

## 单标的流程

```
判定维数 → search_code → 组 rules → frequency+limit → probe → 解读
```

1. **标的**：`search_code`（crypto：`BTCUSDT`）  
2. **规则**：父 playbook 来源表；「上次配置」→ ③ get log 的 `config` / `chart_data.probe`  
3. **`sell_signal`**：单指标/定制 **默认同 `buy_signal`**（否则无卖出）  
4. **limit 优先于 months_back**（服务端 60m 按 7 根/天；crypto 自行算）：

| 场景 | limit |
|------|-------|
| 一般 60m · 1 月 | 154 |
| **Macd4HRhythm** | **≥450**，默认 **462**（3 月） |
| **MACDResonance** | 3 月量级 |
| Crypto 60m | min(24×天, **800**) |

| 策略 | frequency |
|------|-----------|
| Macd4HRhythm | **60m** |
| MACDResonance | **15m**（或 5m） |
| 其他 | catalog |

5. **解读**：买卖次数 + 最近 3 次时间；零信号查 limit/sell/frequency

**registry defaults**：Macd4H — 5/13/9, rhythm 89…；共振 — 12/26/9, zeroAxisRatio 0.002…

---

## 三维 · 本 playbook 覆盖

| 情形 | 覆盖 | 不覆盖 |
|------|------|--------|
| **A 同策略多标的** | ✅ 固定 rules/limit，**循环 code** | — |
| **B 同标的多策略** | ✅ 固定 code，**循环策略**（各 frequency/limit） | — |
| **C 多回测配置** | ⚠️ 可改 limit 看信号条数 | ❌ **盈亏对比** → `strategy-backtest-run` |

### A · 同策略多标的

- 用户：「Macd4H 看腾讯、小米、美团」  
- 步骤：定策略 → 每名称 `search_code` → **最多 4 码** → 循环 probe（共享 buy/sell/limit/frequency）  
- memory：`list(strategy_label)` 看是否已有各 code 的回测信号侧写（optional）  
- 输出：**code | 买次 | 卖次 | 最近触发**

### B · 同标的多策略

- 用户：「腾讯共振 vs 4H 节奏哪个信号多」  
- 步骤：定 code → 每策略组 rules + **专属** frequency/limit → 循环 probe  
- memory：`list(code)` 按 `strategy_label` 参考历史（optional）  
- 输出：**strategy | frequency | 买次 | 卖次 | 最近触发**

### 路由

- 用户要 **收益/回撤/止盈止损** → **`strategy-backtest-run`**（含 A/B/C）  
- 用户要 **旧 log 数字** → **`strategy-backtest-history`**

---

## 配置推断

| 说法 | 推断 |
|------|------|
| 按 UI + Macd4H | 60m · limit≥450 · defaults · sell 镜像 |
| 上次/同样配置 | ③ get log |
| 多标的 / 多策略 | 先识别 **A 或 B**（父 playbook） |

---

## 硬规则 · 反模式

- 必填 code + frequency + buy_signal；禁止 dump bars  
- 禁止 Macd4H months_back=1 或无 sell  
- 禁止情形 C 用 probe 结论「哪套更赚」

## 输出

单次表 · **A/B 并列表** + 一句结论 · 注明参数来源
