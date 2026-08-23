---
name: strategy-signal-probe
description: 策略开发信号测试、probe_bot_signal、probe_bot_signal_series、买卖信号验证、K 线信号序列、定制策略 Macd4HRhythm。用户要「测信号」「有没有买卖点」「策略开发测试」时触发。
skip_retrieval_gate: true
---

# 策略开发 · 信号测试

与 `trading_operation` **策略开发**页的「信号测试」等价，走 GeeGooSignal signal-api。

## 适用 Toolset

`strategy` · `custom_signal`（高级/定制策略）· `market`（`search_code`）

## 标准流程

### 1. 确认标的

`search_code` → 得到 `code`、`name`、`lot_size`

### 2. 组装买卖规则

| 策略类型 | 规则来源 | Agent 动作 |
|----------|----------|------------|
| 单指标 | `get_index_signals` | 展示 name/brief/index/frequency；用户选定后构造 buy 规则 |
| 组合 | `get_signal_combinations` | 展示摘要（name、brief、规则数、indexes）；**选定后**用该项 `buy_signal`/`sell_signal` |
| 高级/定制 | `get_custom_strategy_definitions` → `get_custom_signal_for_skill` | `index` = `strategy_key`；`param` 默认用注册表 `defaults` |

**多匹配时必须 clarify**：`get_signal_combinations` / `get_index_signals` 有 2+ 项都符合用户描述时，调用 `clarify` 列出最多 4 个候选项（名称 + 一句话区别），选定后再组装规则；不要默认猜第一个。

每条规则：`{"index":"MACD","type":"signal","param":{...}}`  
`type` 枚举：**`signal`** | **`flag`**

#### 定制策略 defaults（Monday 注册表）

调用 `get_custom_strategy_definitions` 获取完整 `param_schema`（min/max）。常用默认：

**MACDResonance**（frequency 仅 `15m`/`5m`）

| 参数 | 默认 |
|------|------|
| fastPeriod / slowPeriod / signalPeriod | 12 / 26 / 9 |
| zeroAxisRatio | 0.002 |
| breakoutLookback | 5 |
| enablePseudoCross | true |

**Macd4HRhythm**（frequency 仅 **`60m`**）

| 参数 | 默认 |
|------|------|
| fastPeriod / slowPeriod / signalPeriod | 5 / 13 / 9 |
| rhythmPeriod | 89 |
| zone1Ratio / zone3Ratio | 0.0015 / 0.0045 |
| enableExtremeReversal | true |

用户未指定周期时：**Macd4HRhythm → 60m**；**MACDResonance → 15m**；单指标看 catalog 项 `frequency`。

### 3. 选周期与回溯

| frequency | 典型用途 | UI 默认 period |
|-----------|----------|----------------|
| `5m` / `15m` | 短线、GRID、MACDResonance | 5m 默认 `2w` |
| `60m` | DCA、Macd4HRhythm | **`1m`** |
| `daily` | 趋势 | `1m` |

| 场景 | probe 参数 |
|------|------------|
| 策略开发快速测试 | `limit` 100～300，可不传 `months_back` |
| 对齐回测页 | `months_back` = period 映射（`1m`→1，`2m`→2，`2w`→1） |

`months_back` 工具 schema 默认 **3**；对齐 UI 时优先用 **1**。

### 4. 调用探测

| 场景 | Tool |
|------|------|
| 看整段 K 线上哪些 bar 触发 | `probe_bot_signal_series` |
| 只看当前/指定 bar | `probe_bot_signal`（可选 `at`） |

**最小可用 probe 入参**（用户只说「测一下腾讯 Macd4HRhythm」）：

```json
{
  "code": "00700.HK",
  "frequency": "60m",
  "months_back": 1,
  "buy_signal": [{
    "index": "Macd4HRhythm",
    "type": "signal",
    "param": {
      "fastPeriod": "5", "slowPeriod": "13", "signalPeriod": "9",
      "rhythmPeriod": "89", "zone1Ratio": "0.0015", "zone3Ratio": "0.0045"
    }
  }]
}
```

### 5. 解读结果

- `buy_merged`：`1` = 该 bar 买入；`sell_merged`：`-1` = 卖出
- `buy_rules` / `sell_rules`：每条规则的 `signal_series`、`invalid`、`error`
- **零信号**：检查 frequency 是否在 `supported_frequencies`、规则是否过严、近期是否单边
- 向用户汇报：**买卖次数 + 最近 3 次触发时间**；有 `invalid` 单独列原因

## 配置推断（减少追问）

| 用户说法 | 推断 |
|----------|------|
| 「默认」「常规」「按 UI」 | frequency=60m, months_back=1, 注册表 defaults |
| 「短线」「5 分钟」 | frequency=5m, months_back=1 |
| 「MACD 共振」 | index=MACDResonance, frequency=15m |
| 「4 小时节奏」「市场节奏」 | index=Macd4HRhythm, frequency=60m |
| 已给 `code` + 策略名 | 直接 probe，勿先问周期 |

## 硬规则

- 必须先有 `code` + `frequency` + 至少一条 `buy_signal`
- 定制策略 `index` 必须在 `get_custom_strategy_definitions` 注册表中
- 响应体大（最多 800 bar）；**禁止**全文 dump `bars` / `signal_series`
- 列表 catalog 返回摘要；需要完整规则链时用户选定 `signal_id` 后再取

## 反模式

- 未 `search_code` 就硬编码 code
- 把 `probe` 结果当最终盈亏（PnL 走回测 playbook 或读历史）
- 为「有哪些组合策略」类问题拉取并复述全部 `info` 字段

## 输出

表格：标的、周期、回溯、买入触发次数、卖出触发次数、最近 3 次信号时间；若有 `invalid` 规则单独列出原因。
