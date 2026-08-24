---
name: strategy-backtest
description: 策略回测、网格策略、DCA 定投方案、generate_grid、loopback、信号组合、单指标信号、MACD、SAR、参数建议。用户说「回测」「网格怎么设」「DCA 方案」「哪个信号好」时触发。
---

# 策略生成与回测（父 playbook）

> **子 playbook**（按意图路由，勿混用终态 API）  
> - 测信号 / 有没有买卖点 → **`strategy-signal-probe`**  
> - 跑回测 / 同样参数再跑 / probe 后验证 → **`strategy-backtest-run`**  
> - 看历史 / 上次结果 / log_id → **`strategy-backtest-history`**

## 适用 Toolset

`strategy` · `market`（`search_code`）· `custom_signal`（高级/定制策略）

---

## 参数解析（通用 · 子 playbook 均继承）

组装 **任何** 策略 API 入参（probe、回测、读历史后复跑）时，按下列优先级逐级补全；**上层已有则勿被下层覆盖**。

| 优先级 | 来源 | 何时用 | 典型动作 |
|--------|------|--------|----------|
| **① 本轮明文** | 用户当前消息 | 「腾讯 3 个月 Macd4H 止盈 5%」 | 直接填入对应字段 |
| **② 本会话** | 同会话对话 + Tool 活动 | 刚读过某 `log_id`、刚 probe 过 | 复用会话内 code / log / 规则 |
| **③ 持久 memory** | Mongo 回测历史 · 聊天 recall | 「上次/同样/再跑/那个回测」 | 见下表 |
| **④ 默认表** | catalog · registry · UI 默认 | 全新标的、用户说「按默认/按 UI」 | `get_*` + 注册表 defaults |
| **⑤ clarify** | 各 Gateway 统一 `clarify` | ②③④ 仍歧义或多候选 | 最多 4 项 +「其他」 |

### ③ 持久 memory 怎么取

| 用户意图 | Tool 链 | 取哪些字段 | 供谁用 |
|----------|---------|------------|--------|
| 上次 / 同样 / 再跑 / 那次回测 | `list_strategy_backtest_logs` → `get_strategy_backtest_log` | `code`、`frequency`、`period`、`fund`、`base_order_size`、`config.buy_rules`、`config.sell_rules`、`trade_config` | **backtest-run**（全量）；**signal-probe**（规则 + period→limit） |
| 跨会话、无 log_id | `recall(query=标的/策略关键词)` | 会话摘要中的 code、策略名、log 线索 | 再 `list` 定位 log |
| 仅看结果、不 rerun | 同上 list → get | `result`、`trades` | **backtest-history** |

**list 筛选**：用用户已知的 `code`、`strategy_label`；默认 `limit=5～20`。  
**唯一匹配** → 直接 `get`；**2+ 条** → `clarify` 选 `log_id`（选项含 `created_at` + `profit_rate`）；**0 条** → 降级 ④ 或 `recall`。

**读详情时分区**（同一 `get_strategy_backtest_log`，不同 playbook 取不同块）：

| 区块 | 用途 |
|------|------|
| `run.result` · `trades` | 汇报盈亏（history） |
| `run.config` · `trade_config` · `frequency` · `period` | **复跑 / probe 入参**（run） |
| `run.chart_data.probe` | probe 快照，可还原 probe 请求 |

**param 来源（定制策略）**：probe / 新跑回测默认用 **`get_custom_strategy_definitions` 的 `defaults`**；仅当用户明确「用我保存的那条定制策略」时才用 Mongo `get_custom_signal_for_skill` 的 param。

### 买卖规则来源（通用）

| 策略类型 | Tool | Agent 用法 |
|----------|------|------------|
| 单指标 | `get_index_signals` | 选定 `signal_id` → 该项 `index` + `param` |
| 组合 | `get_signal_combinations` | 选定后整条 **`buy_signal` / `sell_signal`** |
| 高级/定制 | `get_custom_strategy_definitions` | `index` = `strategy_key`；`param` = registry **`defaults`**（见子 playbook 表） |

**多匹配（catalog 2+ 项符合描述）** → 必须先 **`clarify`**（最多 4 项，名称 + 一句区别），勿默认第一个。

每条规则：`{"index":"…","type":"signal"|"flag","param":{…}}`（`param` 用 **数字/布尔**，勿用字符串）。

---

## Gateway · 何时向用户要输入

各端（Web / App / 飞书）均通过 **`clarify(question, choices?)`**；Agent 写法相同，仅展示不同。

| 场景 | 做法 |
|------|------|
| 2～4 个离散方案 | `clarify` + `choices`（UI 自动追加「其他（自行输入）」） |
| 缺一项且无固定枚举 | `clarify` **不传** choices，开放式 |
| 写操作确认（y/n） | **`approval`**，不要用 clarify |
| 飞书 | 用户可回 `A` / `1` / 选项全文 / 自由文本 |
| 无交互环境（cron） | `clarify` 不可用 → 参数须在 ① 给全 |

**必须 clarify 的策略场景**：

- `search_code` 多个合理标的  
- catalog 多个策略匹配  
- `list_strategy_backtest_logs` 多条且用户未指定哪次  
- 历史 param 与用户**新指定**冲突（例：「同样参数但换 BTC」→ 确认是否只改 code）

**不必 clarify**：仅 1 条 list 命中且用户说「上次回测」；本会话刚给出 `log_id`；用户明确「全新配置 / 按默认」。

---

## 路由到子 playbook

| 用户要什么 | 子 playbook | 终态 |
|------------|-------------|------|
| 测信号、买卖点、策略开发测试 | `strategy-signal-probe` | `probe_bot_signal_series` |
| 跑回测、验证收益、同样参数再跑 | `strategy-backtest-run` | **`run_strategy_backtest`**（或 UI 跑 PnL）/ `loopback_strategy` |
| 历史列表、某次盈亏、成交时间线 | `strategy-backtest-history` | list / get |
| 网格 / DCA 方案 + 服务端 loopback | **本 playbook §DCA/GRID** | generate → loopback |

---

## DCA / GRID（本 playbook 直管）

与 Mongo 高级回测 **不同路径**；memory 来自 **`generate_*` 输出**，不是 `strategy_backtest_log`。

1. `search_code`
2. `get_signal_combinations` 或 `get_index_signals` → 选 `signal_id`（未指定时 clarify）
3. `generate_grid_strategy` 或 `generate_dca_strategy`（告知等待：grid ~40s，dca ~2min）
4. `loopback_strategy`：
   - **grid**：`type=grid`，`grid_param` = generate 的 `param`，`frequency=5m`
   - **dca**：`type=dca`，`signal` = generate 的 `buy_signal`，`sl_tp` 来自 `dynamicParam`/`fixedParam`，`frequency=60m`

缺 `fund` / `months_back` → 用 **100000 / 1**（或 ②③ 会话 / recall 中的值）。

---

## 硬规则（通用）

- **禁止**无 `grid_param` / 完整 `signal`+`sl_tp` 裸调 `loopback_strategy`
- **禁止**把 `probe` 买卖次数直接说成收益率（PnL 走回测或 history）
- **禁止**未 `search_code` 硬编码 `code`（crypto 用 `BTCUSDT`，勿用 `BTC.US`）
- 大响应（`bars`、`chart_data`）**禁止**全文 dump
- 列表类 catalog / list API：**列表够用勿逐条 get**（history 列表规则见子 playbook）

## 反模式

- 跳过 ①～③ 直接用 registry 默认，而用户说了「同样/上次」
- 把 `loopback_strategy` 与 trading_operation SmartTrade 回测混为一谈
- 用 clarify 做简单 y/n 写操作确认

## 输出

表格：标的、策略、区间/回溯、信号统计或收益/回撤；注明参数来源（明文 / 历史 log / 默认）；**仅供参考，非投资建议**。
