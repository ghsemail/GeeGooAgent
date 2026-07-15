# Tools 运行态树形图

> **用途**：对话时判断 Tool 是否已注册、默认 chat 能否调用、后端是否真正可用。  
> **代码 SSOT**：`internal/tools/catalog/catalog.go` + `bespoke.go` + `domains.go`  
> **设计全集**：[tool-catalog.md](./tool-catalog.md) · **MCP HTTP**：[interface-map.md](../../../reference/geegoo-mcp/interface-map.md)

---

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 已注册且端到端可用 |
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

### 常踩坑

| Tool | 现象 | 原因 |
|------|------|------|
| `fetch_market_news` / `fetch_stock_news` | `skipped` | 无 script runner |
| `get_ticker` / `get_broker` / `get_position` | trade not configured | 富途 Noop |
| `generate_*_strategy` | 502/404 | analyze-api 未部署 |
| `get_mcp_analysis` | 非旧 LLM 质量 | Analyze/Go 规则化 |
| `loopback_strategy` | 简化模拟 | Signal/Go 确定性回测 |
| `get_capital_*` | A 股 skipped | `.SH`/`.SZ` 显式跳过 |

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
│  ├─ get_ticker, get_broker, get_position       ⚠️ Noop
│  ├─ get_capital_flow, get_capital_distribution ⚠️ A股skip
│  ├─ get_bot_yesterday_attitude                 ✅
│  ├─ get_index_signals, get_signal_combinations ✅ :3210
│  ├─ get_single_prompt_template, get_mcp_analysis ⚠️简化
│  ├─ fetch_market_news, fetch_stock_news        ⚠️ skipped
│  ├─ get_bot_log_by_type                        ✅
│  └─ recall                                     ✅
├─ strategy
│  ├─ generate_grid_strategy, generate_dca_strategy ⚠️
│  └─ loopback_strategy                          ⚠️ :3200
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
| `get_position` / `get_ticker` / `get_broker` | ⚠️ | Noop |
| `generate_grid_strategy` / `generate_dca_strategy` | ⚠️ | Analyze/Go |
| `loopback_strategy` | ⚠️ | :3200 简化回测 |
| 7× Bot/Reminder ×5 | ✅ | 写 Bot 无 scheduler |
| 报告 / Prompt CRUD | ✅ | |

## Bespoke（21）

| Tool | 状态 | chat |
|------|------|------|
| `search_code`, `web_search`, `check_trading_day`, `get_current_price` | ✅ | ✅ |
| `get_report_bot_codes`, `create_pre_market_report`, … | ✅ | 🔒 workflow |
| `fetch_*_news`, `recall_yesterday_summary`, `send_feishu_summary` | ⚠️ | 混合 |
| `get_mcp_analysis`, `get_capital_*` | ⚠️ | ✅ |
| `recall` | ✅ | ✅ |

---

## 对话场景

| 目标 | 推荐 | 避免 |
|------|------|------|
| 查价 | `search_code` → `get_current_price` | `get_ticker` |
| 技术分析 | `get_single_prompt_template` → `get_mcp_analysis` | 缺 `period` |
| DCA 方案 | `get_signal_combinations` → `generate_dca_strategy` | 参数不齐 |
| 新闻 | `web_search` | `fetch_*_news` |
| 盘前写报告 | `/toolsets report_workflow` 或 `geegoo run` | 默认 chat |

---

## 维护

新增 Tool → 改代码 → 同步 **本文件** + [tool-catalog.md](./tool-catalog.md) + [toolsets.md](./toolsets.md)。

*核对：`go test ./internal/tools/... -run TestRegisterAllToolCount`*
