# Tools 运行态树形图

> **用途**：对话时判断 Tool 是否已注册、默认 chat 能否调用、后端是否真正可用。  
> **代码 SSOT**：`internal/tools/catalog/catalog.go` + `bespoke.go` + `domains.go`  
> **设计全集**：[tool-catalog.md](./tool-catalog.md) · **MCP HTTP**：[interface-map.md](../../../reference/geegoo-mcp/interface-map.md)

---

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 已注册且端到端可用 |
| 💬 | 接口正常，需**引导用户选择**参数后再调（先澄清意图，非故障） |
| ⚠️ | 已注册但部分可用（Stub / Noop / 简化 / 环境缺失） |
| ❌ | 未注册到 Registry |
| 🔒 | 默认 chat 白名单不含 |

实现层：**Agent/Go** · **Bot/Go** :3120 · **Signal/Go** :3200/3210 · **Analyze/Go** :3230 · **Data/Go** · **Script** · **Stub**

> GeeGoo 栈禁止转发旧 Trading Python（5600/5700）。

---

## 总览

| 维度 | 数量 |
|------|------|
| Registry 已注册 | **82** |
| 默认 chat 白名单 | **~73** |

### 需引导用户选择（💬）

| Tool | Agent 应先做什么 |
|------|------------------|
| `generate_dca_strategy` | 问单指标/组合 → `get_index_signals` 或 `get_signal_combinations` 展示 brief → 用户选 `signal_id` |
| `loopback_strategy` | grid：先 `generate_grid_strategy` 或确认 `grid_param`；dca：确认 `signal` + `sl_tp` |
| `get_mcp_analysis` | 确认 `period`（及 `prompt_id`）；`get_single_prompt_template` 取模板 |
| `get_bot_yesterday_attitude` | 先 list 机器人，让用户指定 `bot_id` |
| `get_stock_daily_reports` | 确认 `report_date`（`YYYY-MM-DD`） |
| `create_*_report` 等 workflow | 确认 `stock_name` / `code` 等；默认 chat 不含 🔒 |

### 常踩坑（⚠️）

| Tool | 现象 | 原因 |
|------|------|------|
| `fetch_market_news` / `fetch_stock_news` | 极少 skip | Go RSS/东财回退（无 Python 也可用） |
| `generate_grid_strategy` | 未配 prompt 时为启发式 | 需 `GEEGOO_PROMPT_API_URL` + configured LLM |
| `get_mcp_analysis` | 非旧 LLM 质量 | Analyze/Go 规则化（可用） |
| `get_capital_*` | A 股 MCP 空时 | Agent 东财回退；HK 走 GeeGooData |
| `send_feishu_summary` | skip | 未配 `feishu_webhook_url` |

### 默认 chat toolset

| ID | 默认 chat | 工具数 |
|----|-----------|--------|
| `market` | ✅ | 17 |
| `strategy` | ✅ | 3 |
| `bot_manager` | ✅ | 20 |
| `reminder_manager` | ✅ | 15 |
| `report_query` | ✅ | 10 |
| `report_workflow` | ❌ 🔒 | 8 |

切换：`/toolsets market,strategy`

---

## 树形图（82 registered）

```
GeeGooAgent Tools
├─ market [toolset]
│  ├─ search_code, web_search                    ✅
│  ├─ check_trading_day, get_current_price       ✅
│  ├─ get_ticker, get_broker, get_position       ✅ futu_bridge
│  ├─ get_capital_flow, get_capital_distribution ⚠️ A股 MCP 空→Agent 东财
│  ├─ get_bot_yesterday_attitude                 💬 需 bot_id
│  ├─ get_index_signals, get_signal_combinations ✅ :3210
│  ├─ get_single_prompt_template                 ✅
│  ├─ get_mcp_analysis                           💬 需 period；⚠️ 规则化输出
│  ├─ fetch_market_news, fetch_stock_news        ✅ Go/Python 回退
│  ├─ get_bot_log_by_type                        ✅
│  └─ recall                                     ✅
├─ strategy
│  ├─ generate_grid_strategy                        ✅ LLM（配 prompt）；否则启发式
│  ├─ generate_dca_strategy                           💬 先选 signal_id；LLM 同 grid
│  └─ loopback_strategy                               💬 grid 需 grid_param；dca 需 signal
├─ bot_manager（4×5 CRUD+log）
├─ reminder_manager（3×5）
├─ report_query（盘前/盘中/盘后 CRUD + 聚合）
├─ report_workflow 🔒（create_pre_market, save_local, …）
└─ prompt_template（竞品/ETF 模板 CRUD×6）
```

完整展开见下文 §HTTP / §Bespoke。

---

## HTTP 转发（62）

| 后端 | Tools |
|------|-------|
| Bot :3120 | 默认全部 HTTP |
| Signal :3200 | `loopback_strategy` |
| Catalog :3210 | `get_index_signals`, `get_signal_combinations` |

| Tool | 状态 | 备注 |
|------|------|------|
| `get_position` / `get_ticker` / `get_broker` | ✅ | futu_bridge |
| `generate_grid_strategy` | ✅ | Analyze/Go；配 prompt+LLM 为 AIServer 级 |
| `generate_dca_strategy` | 💬 | 先问单指标/组合 → 用户选 `signal_id`；LLM 同 grid |
| `loopback_strategy` | 💬 | grid 需 `grid_param`；dca 需 `signal` 链（:3200 K 线回测） |
| 7× Bot/Reminder ×5 | ✅ | 写 Bot 无 scheduler |
| 报告 / Prompt CRUD | ✅ | |

## Bespoke（21）

| Tool | 状态 | chat |
|------|------|------|
| `search_code`, `web_search`, `check_trading_day`, `get_current_price` | ✅ | ✅ |
| `get_report_bot_codes`, `create_pre_market_report`, … | 💬/🔒 | workflow；缺参时先向用户确认 |
| `fetch_*_news`, `recall_yesterday_summary` | ✅ | 新闻 Go 回退；昨日摘要无文件时 skip |
| `send_feishu_summary` | ⚠️ | 需 `feishu_webhook_url` |
| `get_mcp_analysis` | 💬/⚠️ | 💬 缺 `period`；⚠️ 规则化分析 |
| `get_capital_*` | ⚠️ | A 股东财回退 |
| `recall` | ✅ | ✅ |

---

## 对话场景

| 目标 | 推荐 | 避免 |
|------|------|------|
| 查价 | `search_code` → `get_current_price` | `get_ticker` |
| 技术分析 | `get_single_prompt_template` → `get_mcp_analysis` | 缺 `period` |
| DCA 方案 | 先问单指标/组合 → `get_index_signals` 或 `get_signal_combinations` 展示 brief → 用户选 `signal_id` → `generate_dca_strategy` | 未选信号就调 generate；参数不齐 |
| 新闻 | `web_search` | `fetch_*_news` |
| 盘前写报告 | `/toolsets report_workflow` 或 `geegoo run` | 默认 chat |

---

## 维护

新增 Tool → 改代码 → 同步 **本文件** + [tool-catalog.md](./tool-catalog.md) + [toolsets.md](./toolsets.md)。

*核对：`go test ./internal/tools/... -run TestRegisterAllToolCount`*
