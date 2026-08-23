---
name: strategy-signal-probe
description: 策略开发信号测试、probe_bot_signal、probe_bot_signal_series、买卖信号验证、K 线信号序列、定制策略 Macd4HRhythm。用户要「测信号」「有没有买卖点」「策略开发测试」时触发。
---

# 策略开发 · 信号测试

与 `trading_operation` **策略开发**页的「信号测试」等价，走 GeeGooSignal signal-api。

## 适用 Toolset

`strategy` · `custom_signal`（高级/定制策略）· `market`（`search_code`）

## 标准流程

### 1. 确认标的

`search_code` → 得到 `code`、`name`、`lot_size`

### 2. 组装买卖规则

| 策略类型 | 规则来源 |
|----------|----------|
| 单指标 | `get_index_signals` → 取 `buy_signal` / `sell_signal` 模板 |
| 组合 | `get_signal_combinations` → 整条 buy/sell 链 |
| 高级/定制 | `get_custom_signal_for_skill` + `get_custom_strategy_definitions` → `index` = `custom.index` |

每条规则：`{"index":"MACD","type":"signal","param":{...}}`

### 3. 选周期与回溯

| frequency | 典型用途 |
|-----------|----------|
| `5m` / `15m` | 短线、GRID |
| `60m` | DCA、共振类 |
| `daily` | 趋势 |

- **策略开发**：只传 `limit`（30～800），可不传 `months_back`
- **对齐回测页**：传 `months_back`（如 1、3）+ `limit`

### 4. 调用探测

| 场景 | Tool |
|------|------|
| 看整段 K 线上哪些 bar 触发 | `probe_bot_signal_series` |
| 只看当前/指定 bar | `probe_bot_signal`（可选 `at`） |

### 5. 解读结果

- `buy_merged`：`1` = 该 bar 买入信号；`sell_merged`：`-1` = 卖出
- `buy_rules` / `sell_rules`：每条规则的 `signal_series`、`value_series`、`reasons`、`invalid`、`error`
- **零信号**：检查规则是否过严、周期是否匹配 `supported_frequencies`、标的近期是否单边

## 硬规则

- 必须先有 `code` + `frequency` + 至少一条 `buy_signal`
- 定制策略 `index` 必须在 `get_custom_strategy_definitions` 注册表中
- 响应体大（最多 800 bar）；向用户汇报时**摘要**买卖次数与最近几次触发时间，勿全文 dump

## 反模式

- 未 `search_code` 就硬编码 code
- 把 `probe` 结果当最终盈亏（PnL 需走回测 playbook 或读历史）

## 输出

表格：标的、周期、回溯、买入触发次数、卖出触发次数、最近 3 次信号时间；若有 `invalid` 规则单独列出原因。
