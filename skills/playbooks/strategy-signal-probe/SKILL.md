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
判定维数 → search_code → 组 rules → frequency + 回溯时长 → probe → 解读
```

> **不测能否买入。** 可买性（fund / lot_size / base_order_size）只在 **`strategy-backtest-run`** 跑 PnL 前预检；本 playbook 只管信号。

1. **标的**：`search_code`（crypto：`BTCUSDT`）
2. **规则**：父 playbook 来源表；「上次配置」→ ③ get log 的 `config` / `chart_data.probe`  
3. **`sell_signal`**：单指标/定制 **默认同 `buy_signal`**（否则无卖出）  
4. **回溯时长（优先 `months_back`，不要手算 `limit`）**：

服务端逻辑：`limit` 未传时按 `months_back` + `frequency` 自动换算 K 线根数；**只有**要对齐 Web「1天/1周」等细粒度，或 Macd4H 需强制下限且不想改月数时，才显式传 `limit`。

| 用户 / UI 说法 | probe 传参 |
|----------------|------------|
| 默认 / 未说明 | **`months_back: 3`**（服务端默认；Macd4H/共振够用） |
| 1 个月 / UI「1月」 | **`months_back: 1`** |
| 3 个月 / UI「3月」 | **`months_back: 3`** |
| Macd4HRhythm | **`frequency: 60m`** + **`months_back: 3`**（≈462 根，勿用 months_back=1） |
| MACDResonance | **`frequency: 15m`**（或 5m）+ **`months_back: 3`** |

| 策略 | frequency |
|------|-----------|
| Macd4HRhythm | **60m** |
| MACDResonance | **15m**（或 5m） |
| 其他单指标 | catalog 默认（通常 60m 或 daily） |

5. **解读**：优先读 tool **`summary`**（含 `buy_hits`/`sell_hits`）及 compact 后的 `recent_*_times`；禁止 dump 全量 `bars`  
6. **零信号**：查 `months_back` / `sell_signal` / `frequency`（不是 limit 算错）

**registry defaults**：Macd4H — 5/13/9, rhythm 89…；共振 — 12/26/9, zeroAxisRatio 0.002…

---

## 三维 · 本 playbook 覆盖

| 情形 | 覆盖 | 不覆盖 |
|------|------|--------|
| **A 同策略多标的** | ✅ 固定 rules/**months_back**，**循环 code** | — |
| **B 同标的多策略** | ✅ 固定 code，**循环策略**（各 frequency/**months_back**） | — |
| **C 多回测配置** | ⚠️ 可改 **months_back** 看信号条数 | ❌ **盈亏对比** → `strategy-backtest-run` |

### A · 同策略多标的

- 用户：「Macd4H 看腾讯、小米、美团」  
- 步骤：定策略 → 每名称 `search_code` → **最多 4 码** → 循环 probe（共享 buy/sell/**months_back**/frequency）  
- memory：`list(strategy_label)` 看是否已有各 code 的回测信号侧写（optional）  
- 输出：**code | 买次 | 卖次 | 最近触发**

### B · 同标的多策略

- 用户：「腾讯共振 vs 4H 节奏哪个信号多」  
- 步骤：定 code → 每策略组 rules + **专属** frequency/**months_back** → 循环 probe  
- memory：`list(code)` 按 `strategy_label` 参考历史（optional）  
- 输出：**strategy | frequency | 买次 | 卖次 | 最近触发**

### 路由

- 用户要 **收益/回撤/止盈止损** → **`strategy-backtest-run`**（含 A/B/C）  
- 用户要 **旧 log 数字** → **`strategy-backtest-history`**

---

## 配置推断

| 说法 | 推断 |
|------|------|
| 按 UI 默认 | **months_back: 3**（或用户说的 1月→1）+ catalog frequency + sell 镜像 |
| 按 UI + Macd4H | **60m** + **months_back: 3** + defaults + sell 镜像 |
| 上次/同样配置 | ③ get log |
| 多标的 / 多策略 | 先识别 **A 或 B**（父 playbook） |

---

## 单指标歧义（RSI 等）

GeeGooSignal catalog 里 RSI 常见 3 条：**阈值信号**、**金死叉信号**（买卖交叉）、**阈值趋势**（type=flag，只判多空，**不能**当 probe/回测买卖点）。

| 用户说法 | 选用 |
|----------|------|
| RSI / RSI策略 / 测RSI / Eval「RSI阈值信号」 | catalog **RSI阈值信号**（type=signal） |
| RSI金死叉 / RSI交叉 / 快慢线 | **RSI金死叉信号**（index RSICROSS） |
| RSI趋势 / 只看多空 | **RSI阈值趋势**（type=flag；仅用户明确要 flag 时用） |

- `get_index_signals` 按「RSI」仍命中 **2+ 条**且用户**未**指定上表、会话也**还没有**已选信号 → **必须** `clarify(question, choices=[最多 4 个 catalog name])` 并停等；已选过则沿用。**禁止**只在正文写「请选择 A/B/C」——Web **只有 clarify 工具**才会出现选项按钮。**禁止**每轮对话重复选信号。
- 信号测试 / 回测：**禁止**默认 type=flag 的「阈值趋势」。

---

## 硬规则 · 反模式

- 必填 code + frequency + buy_signal；**默认只传 `months_back`，禁止手算 limit**；禁止 dump bars  
- 禁止 Macd4H **months_back=1** 或无 sell  
- 禁止情形 C 用 probe 结论「哪套更赚」

## 输出

单次表 · **A/B 并列表** + 一句结论 · 注明参数来源
