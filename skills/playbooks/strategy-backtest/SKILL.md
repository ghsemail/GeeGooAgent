---
name: strategy-backtest
description: 策略回测、网格/DCA、信号测试、多标的对比、多策略对比、回测参数对比、generate_grid、loopback、MACD。用户说「回测」「测信号」「哪个策略好」「这几只一起测」「同样配置不同止盈」时触发。
---

# 策略生成与回测（父 playbook）

> **子 playbook**  
> | 意图 | Playbook | 终态 Tool |
> |------|----------|-----------|
> | 测信号 / 买卖点 | **`strategy-signal-probe`** | `probe_bot_signal_series` |
> | 跑回测 / 验证收益 | **`strategy-backtest-run`** | **`run_strategy_backtest`** / `loopback_strategy` |
> | 看历史 / 对比旧 log | **`strategy-backtest-history`** | `list_strategy_backtest_logs` / `get_strategy_backtest_log` |
> | 网格 / DCA loopback | **本 playbook §DCA/GRID** | `generate_*` → `loopback_strategy` |

## 适用 Toolset

`strategy` · `market`（`search_code`）· `custom_signal`

---

## 三维场景 · 覆盖矩阵

用户一次请求常只变 **一个维度**（A / B / C）。先判定变维，再选 playbook；**混合多维**（如同时换标的又换止盈）→ **clarify** 确认先比哪一维。

| 情形 | 固定 | 变化 | Probe | 回测 PnL | 读历史 | list 分组 memory |
|------|------|------|-------|----------|--------|------------------|
| **A · 同策略多标的** | 策略 rules · frequency · limit · trade_config | **`code[]`** | ✅ 循环 probe | ✅ 循环 `run_strategy_backtest` | ✅ 按 code 并列 | `list(strategy_label=…)` **不设 code** → 按 `code` 分组 |
| **B · 同标的多策略** | **`code`** | **策略 rules**（frequency/limit 各不同） | ✅ 按策略循环 | ✅ 按策略循环 | ✅ 按策略并列 | `list(code=…)` **不设 strategy_label** → 按 `strategy_label` 分组 |
| **C · 同标的同策略多配置** | **`code` + rules** | period · trade_config · fund 等 | ⚠️ 仅看信号密度（不比 PnL） | ✅ **必须** 循环 run 或读 log | ✅ **主路径** | `list(code=…, strategy_label=…)` → 按 `period`/时间 对比 |

**list 摘要已含**：`code`、`strategy_label`、`period`、`frequency`、`result.profit_rate`、`result.drawdown`、`trade_count` — **情形 A/B/C 的粗对比 often 无需 get**。  
**list 不含** `trade_config` 细节；比止盈/止损差异 → 对选中的 2～4 条 **get** 取 `trade_config`，或看 `period` + 用户已知改动。

---

## 参数解析（通用 · 子 playbook 均继承）

| 优先级 | 来源 | 动作 |
|--------|------|------|
| **① 明文** | 当前消息 | 直接填 |
| **② 会话** | 本会话 Tool/对话 | 复用 log_id、code、规则 |
| **③ memory** | list → get · `recall` | 见下 |
| **④ 默认** | catalog / registry / UI 默认 | `get_*` + defaults |
| **⑤ clarify** | 歧义 / >4 项 | 最多 4 项 +「其他」 |

### ③ memory（含三维）

| 意图 | Tool | 取什么 |
|------|------|--------|
| 上次 / 同样 / 再跑 | list → get | 全量 `config` + `trade_config` + `period` |
| **A** 同策略曾测多码 | `list(strategy_label)` | 各 `code` 最新一条的 `result` |
| **B** 同码曾测多策略 | `list(code)` | 各 `strategy_label` 最新一条 |
| **C** 同码同策略多套配置 | `list(code, strategy_label)` | 各 `period`/日期的 `result`；细比 get |
| 跨会话 | `recall` | 线索 → 再 list |

**单 log 复跑**：唯一匹配直接 get；2+ 条 clarify（`created_at` + `profit_rate` + `period`）。

**定制 param**：默认 registry **`defaults`**；仅用户指定「我保存的那条」才用 `get_custom_signal_for_skill`。

### 买卖规则来源

| 类型 | Tool | 用法 |
|------|------|------|
| 单指标 | `get_index_signals` | `signal_id` → index + param |
| 组合 | `get_signal_combinations` | 整条 buy/sell 链 |
| 高级/定制 | `get_custom_strategy_definitions` | `strategy_key` + **defaults** |

catalog 2+ 匹配 → **clarify**。规则 JSON：`type`=`signal`|`flag`，**param 用数字/布尔**。

---

## 三维 · 执行要点

**批量上限**：默认最多 **4** 个变化项；超出 clarify 选子集或分批。  
**公平对比**：情形 B **禁止**强行统一 frequency（Macd4H=60m vs 共振=15m 是正常差异）；除非用户明确要求「都用 60m」。

### A · 同策略多标的

1. 固定策略 → rules + frequency + limit (+ trade_config)。  
2. 每个名称 `search_code`；歧义 clarify。  
3. **循环**（只改 `code`）：`probe_bot_signal_series` 或 `run_strategy_backtest`。  
4. memory：优先 `list(strategy_label)` 已有结果；新跑用同一套 trade_config。  
5. 输出列：`code` | 买卖次数或 profit_rate | 回撤 | log_id。

### B · 同标的多策略

1. `search_code` → 唯一 `code`。  
2. 每个策略单独 rules + **该策略** frequency + **该策略** limit（见 signal-probe 表）。  
3. MACDResonance → 可 `macd_resonance_v1` trade_config；Macd4H → generic_smarttrade。  
4. memory：`list(code)` 按 `strategy_label` 取已有收益。  
5. 输出列：`strategy_label` | frequency | 信号或收益指标。

### C · 同标的同策略多配置（仅回测）

1. 固定 `code` + buy/sell 链（① 或 get 一条基准 log）。  
2. 列出 2～N 套差异：

| 轴 | 字段 |
|----|------|
| 回溯 | `period` · `limit` |
| 止盈止损 | `trade_config.tp` · `trade_config.sl` |
| 资金/仓位 | `fund` · `base_order_size` · `position` |
| 执行 | `execution_profile` · `macd_exec` |

3. **有历史**：`list(code, strategy_label)` → 列表比 `period`+`profit_rate`；比 TP/SL → get 2～4 条。  
4. **新跑**：循环 `run_strategy_backtest`，每次只改 config 块；返回多个 `log_id`。  
5. **配置摘要行**（输出用）：`{period} · TP{fix_tp}% · SL{sl_mode}/{index} · fund`  
6. probe 改 limit 仅说明「信号条数差异」，**不得**替代 PnL 对比。

---

## Gateway · clarify

| 触发 | 示例问题 |
|------|----------|
| 变维不明 | 「要比标的、策略还是回测参数？」 |
| A 标的 >4 | 「先测哪 4 只？」 |
| B 策略 >4 | 「先比哪几个策略？」 |
| C 多套 config | 「比 1m/3m 还是比 TP5%/7%？」 |
| list 多条 | 「选哪次回测？」（带 date + profit_rate + period） |

Web/App/飞书均用 **`clarify(choices?)`**；y/n 写操作用 **`approval`**。

---

## DCA / GRID

memory 来自 **`generate_*`**，非 `strategy_backtest_log`。缺 fund/months_back → **100000 / 1**。详见原流程：`search_code` → 选 signal → `generate_*` → `loopback_strategy`。

---

## 硬规则

- 禁止 probe 买卖次数当收益率；禁止裸调 `loopback_strategy`  
- 禁止未 `search_code` 硬编码（crypto：`BTCUSDT`）  
- 禁止 dump 全量 bars/chart_data  
- 情形 C 禁止仅用 probe 断言「哪套配置更赚」

## 输出

- **单次**：标的 · 策略 · 回溯 · 信号或收益 · 参数来源  
- **A/B/C 对比**：并列表 + 一句结论 · **仅供参考，非投资建议**
