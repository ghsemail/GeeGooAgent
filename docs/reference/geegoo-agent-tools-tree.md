# GeeGooAgent Tools 完整树形图

> **用途**：与 GeeGooAgent 对话时，快速判断某个能力是否已注册、默认 chat 能否调用、后端是否真正可用。  
> **代码 SSOT**：`internal/tools/catalog/catalog.go`（HTTP 转发）+ `internal/tools/bespoke.go`（本地/定制）+ `internal/tools/domains.go`（分组）  
> **后端 SSOT**：GeeGooBot [`implemented-routes.md`](../../../GeeGooBot/docs/api/implemented-routes.md) · GeeGooSignal analyze/signal/catalog  
> **生成基准**：2026-07-14（本地仓库状态）  
> **架构导读**：[../architecture/README.md](../architecture/README.md) · [tools-and-skills.md](../architecture/tools-and-skills.md)

---

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 已注册且端到端可用（需 `mcp_token` / 服务在线等前置条件满足） |
| ⚠️ | 已注册但**部分可用**：Stub、Noop、简化实现、环境缺失、A 股跳过、线上未部署等 |
| ❌ | **未注册**到 Agent Registry，对话中无法调用（即便文档/Skill 提到过） |
| 🔒 | 默认 **chat 白名单不含**（仅 workflow / 手动加 toolset 时可用） |

### 实现层缩写

| 层 | 说明 |
|----|------|
| **Agent/Go** | GeeGooAgent 本地 handler（`bespoke.go` 等） |
| **Bot/Go** | GeeGooBot `mcp-api` :3120，原生 Go |
| **Signal/Go** | GeeGooSignal `signal-api` :3200 / `catalog-api` :3210 |
| **Analyze/Go** | GeeGooSignal `analyze-api` :3230 |
| **Data/Go** | GeeGooData 行情/资金接口 |
| **Script** | 捆绑 Python/Skill 脚本（当前 Agent 未接 runner） |
| **Stub** | 注册但固定返回 `skipped` / 未实现 |

> **架构原则**：GeeGoo 栈**禁止** HTTP 转发旧 Trading Python（5600/5700）。下文凡标注 Signal/Analyze/Data 均为 **Go 原生**。

---

## 一、总览统计

| 维度 | 数量 | 说明 |
|------|------|------|
| Registry 已注册 | **82** | HTTP 62 + bespoke 21（`search_code` 双份定义，bespoke 覆盖 HTTP） |
| 默认 chat 白名单 | **~73** | 5 个默认 toolset，不含 `report_workflow`；`switch_bot` 在白名单定义中但**未注册** |
| GeeGooBot mcp-api 路由 | **73** | 与 Agent HTTP 工具基本一一对应（无 Agent 侧的 `/getDailyPriceChange` 等） |
| 规划未注册 | **11** | 见 [§九 规划未注册](#九规划未注册仅文档--skill-提及) |
| 对话常踩坑 | **8** | 见下表 |

### 对话时「注册了但用不了 / 效果打折」速查

| Tool | 现象 | 原因 |
|------|------|------|
| `fetch_market_news` / `fetch_stock_news` | `skipped` | Script runner 未接入，非 dry-run 直接跳过 |
| `get_ticker` / `get_broker` / `get_position` | 报错 `trade provider not configured` | Bot/Go **Noop**，富途未接线 |
| `generate_dca_strategy` / `generate_grid_strategy` | 502 / 404 | 线上 analyze-api 若未部署原生路由会失败（本地代码已有 Analyze/Go） |
| `get_mcp_analysis` | 有结果但非旧版 LLM 质量 | Analyze/Go 规则化报告，`model=GeeGooSignal-native` |
| `loopback_strategy` | 有结果但是简化模拟 | Signal/Go 确定性回测，非真实历史撮合 |
| `get_capital_flow` / `get_capital_distribution` | A 股 `skipped` | 代码显式跳过 `.SH` / `.SZ` |
| `recall_yesterday_summary` | `skipped` | Stub， episodic 记忆未实现 |
| `send_feishu_summary` | `skipped` | 未配置 webhook |
| `switch_bot` | **工具不存在** | domains 规划了，Registry **未注册**，Bot 也无 `/switchBot` |

### 默认 chat toolset

| ID | 中文 | 默认进 chat | 工具数（定义） |
|----|------|-------------|----------------|
| `market` | 行情与分析 | ✅ | 17 |
| `strategy` | 策略生成与回测 | ✅ | 3 |
| `bot_manager` | 交易 Bot | ✅ | 21（含未注册的 `switch_bot`） |
| `reminder_manager` | 提醒 Bot | ✅ | 15 |
| `report_query` | 报告查询 | ✅ | 10 |
| `report_workflow` | 报告 Workflow | ❌ 🔒 | 8 |

切换：`/toolsets market,strategy` · 恢复默认：`/toolsets default`

---

## 二、树形图（按业务域）

```
GeeGooAgent Tools (82 registered)
│
├─ 📊 行情与分析 [toolset: market] ─────────────────────────────────────
│  ├─ 标的与检索
│  │  ├─ search_code              ✅ Agent/Go → Signal/Go :3200 /searchCode
│  │  └─ web_search               ✅ Agent/Go（DuckDuckGo，需网络）
│  ├─ 交易日与现价
│  │  ├─ check_trading_day        ✅ Agent/Go → Bot/Go → Data/Go
│  │  └─ get_current_price        ✅ Agent/Go → Bot/Go → Data/Go
│  ├─ 盘中深度行情（富途）
│  │  ├─ get_ticker               ⚠️ Bot/Go → trade.Noop（未配置）
│  │  ├─ get_broker               ⚠️ 同上
│  │  └─ get_position             ⚠️ 同上
│  ├─ 资金与态度
│  │  ├─ get_capital_flow         ⚠️ Agent/Go → Data/Go；A 股 skipped
│  │  ├─ get_capital_distribution ⚠️ 同上
│  │  └─ get_bot_yesterday_attitude ✅ Agent/Go → Bot/Go（Mongo 报告）
│  ├─ 信号目录
│  │  ├─ get_index_signals        ✅ HTTP → Signal/Go catalog :3210
│  │  └─ get_signal_combinations  ✅ HTTP → Signal/Go catalog :3210
│  ├─ 技术分析
│  │  ├─ get_single_prompt_template ✅ Agent/Go → Bot/Go → catalog-api
│  │  └─ get_mcp_analysis         ⚠️ Agent/Go → Bot/Go → Analyze/Go（规则化，非完整 LLM）
│  ├─ 新闻（本地脚本）
│  │  ├─ fetch_market_news        ⚠️ Stub/Script（runner 不可用 → skipped）
│  │  └─ fetch_stock_news         ⚠️ Stub/Script（同上）
│  ├─ 日志
│  │  └─ get_bot_log_by_type      ✅ HTTP → Bot/Go（Mongo）
│  └─ 会话记忆
│     └─ recall                   ✅ Agent/Go（本地 session 检索）
│
├─ 📈 策略生成与回测 [toolset: strategy] ─────────────────────────────────
│  ├─ generate_grid_strategy      ⚠️ HTTP → Bot/Go → Analyze/Go（指标规则，非 LLM）
│  ├─ generate_dca_strategy       ⚠️ 同上；线上需确认 analyze-api 已部署
│  └─ loopback_strategy           ⚠️ HTTP → Signal/Go :3200（简化确定性回测）
│
├─ 🤖 交易 Bot [toolset: bot_manager] ───────────────────────────────────
│  ├─ DCA 交易 Bot（各 5：create/update/delete/list/get_*_log）
│  │  ├─ create_dca_bot           ✅ HTTP → Bot/Go（Mongo 写入；⚠️ 无 scheduler 自动跑）
│  │  ├─ update_dca_bot           ✅
│  │  ├─ delete_dca_bot           ✅
│  │  ├─ list_dca_bots            ✅
│  │  └─ get_dca_bot_log          ✅
│  ├─ GRID 交易 Bot
│  │  ├─ create_grid_bot          ✅（同上备注）
│  │  ├─ update_grid_bot          ✅
│  │  ├─ delete_grid_bot          ✅
│  │  ├─ list_grid_bots           ✅
│  │  └─ get_grid_bot_log         ✅
│  ├─ SmartTrade
│  │  ├─ create_smart_trade       ✅
│  │  ├─ update_smart_trade       ✅
│  │  ├─ delete_smart_trade       ✅
│  │  ├─ list_smart_trades        ✅
│  │  └─ get_smart_trade_log      ✅
│  ├─ HDG 对冲
│  │  ├─ create_hdg_bot           ✅
│  │  ├─ update_hdg_bot           ✅
│  │  ├─ delete_hdg_bot           ✅
│  │  ├─ list_hdg_bots            ✅
│  │  └─ get_hdg_bot_log          ✅
│  └─ switch_bot                  ❌ 未注册（Bot 无 /switchBot 路由）
│
├─ 🔔 提醒 Bot [toolset: reminder_manager] ──────────────────────────────
│  ├─ DCA 提醒（5 CRUD + log）
│  │  ├─ create_dca_reminder      ✅ HTTP → Bot/Go
│  │  ├─ update_dca_reminder      ✅
│  │  ├─ delete_dca_reminder      ✅
│  │  ├─ list_dca_reminders       ✅
│  │  └─ get_dca_reminder_log     ✅
│  ├─ GRID 提醒
│  │  ├─ create_grid_reminder     ✅
│  │  ├─ update_grid_reminder     ✅
│  │  ├─ delete_grid_reminder     ✅
│  │  ├─ list_grid_reminders      ✅
│  │  └─ get_grid_reminder_log    ✅
│  └─ Smart 提醒
│     ├─ create_smart_reminder    ✅
│     ├─ update_smart_reminder    ✅
│     ├─ delete_smart_reminder    ✅
│     ├─ list_smart_reminders     ✅
│     └─ get_smart_reminder_log   ✅
│
├─ 📄 报告查询 [toolset: report_query] ──────────────────────────────────
│  ├─ 盘前
│  │  ├─ update_pre_market_report ✅ HTTP → Bot/Go
│  │  ├─ delete_pre_market_report ✅
│  │  └─ get_pre_market_reports   ✅
│  ├─ 盘中
│  │  ├─ create_intraday_report   ✅
│  │  ├─ update_intraday_report   ✅
│  │  ├─ delete_intraday_report   ✅
│  │  └─ get_intraday_reports     ✅
│  ├─ 盘后
│  │  ├─ create_post_market_report ✅
│  │  ├─ update_post_market_report ✅
│  │  ├─ delete_post_market_report ✅
│  │  └─ get_post_market_reports  ✅
│  └─ 聚合查询
│     ├─ get_stock_daily_reports  ✅ Agent/Go → Bot/Go
│     └─ list_today_reports       ✅ Agent/Go（幂等检查别名）
│
├─ 🔄 报告 Workflow [toolset: report_workflow] 🔒 默认不进 chat ─────────
│  ├─ get_report_bot_codes        ✅ Agent/Go → Bot/Go
│  ├─ create_pre_market_report    ✅ Agent/Go → Bot/Go（写操作需 interactive 审批）
│  ├─ save_local_report           ✅ Agent/Go（工作区本地 Markdown）
│  ├─ write_execution_log         ✅ Agent/Go（工作区 execution-log.md）
│  ├─ read_working_state          ✅ Agent/Go（WorkingMemory，需配置 store）
│  ├─ recall_yesterday_summary    ⚠️ Stub → skipped
│  ├─ send_feishu_summary         ⚠️ Stub → skipped（无 webhook）
│  └─ get_bot_yesterday_attitude  （亦在 market；workflow 盘前常用）
│
├─ 📝 Prompt 模板 [prompt_template 域] ───────────────────────────────────
│  ├─ create_competitor_prompt_template  ✅ HTTP → Bot/Go → catalog-api
│  ├─ edit_competitor_prompt_template    ✅
│  ├─ delete_competitor_prompt_template  ✅
│  ├─ create_etf_prompt_template         ✅
│  ├─ edit_etf_prompt_template           ✅
│  └─ delete_etf_prompt_template         ✅
│
└─ 📋 规划未注册（仅文档 / Skill 提及）──────────────────────────────────
   ├─ switch_bot                 ❌ Bot 路由 + Agent 均未实现
   ├─ wait_for_human             ❌ Bot 创建前人工确认
   ├─ spawn_subagent             ❌ 子 Agent 编排
   ├─ fetch_global_quote         ❌ 免费全球行情脚本
   ├─ get_tech_prompt_list       ❌ 遗留名；用 get_single_prompt_template
   ├─ recall_past_attitude       ❌ attitude_history.jsonl
   ├─ recall_similar_setup       ❌ 向量语义记忆
   ├─ compare_daily_reports      ❌ 报告差分
   ├─ get_daily_reports_unified  ❌ reportServer :6100
   ├─ update_working_state       ❌ 一般由 Runtime 写，非 LLM 直调
   └─ emit_event                 ❌ 调试 EventBus
```

---

## 三、按注册类型展开

### 3.1 HTTP 转发工具（62 个）

由 `RegisterHTTPFromCatalog` 注册，经 `HTTPBackends.ForTool` 选后端：

| 后端 | Tools |
|------|-------|
| **Bot/Go :3120**（默认） | 除下表外全部 HTTP 工具 |
| **Signal/Go :3200** | `search_code`（bespoke 覆盖）, `loopback_strategy` |
| **Signal catalog :3210** | `get_index_signals`, `get_signal_combinations` |

完整列表（HTTP path → 状态）：

| Tool | MCP HTTP | 状态 | 实现 |
|------|----------|------|------|
| `get_position` | `/getPosition` | ⚠️ | Bot/Go → Noop |
| `get_ticker` | `/getTicker` | ⚠️ | Bot/Go → Noop |
| `get_broker` | `/getBroker` | ⚠️ | Bot/Go → Noop |
| `get_index_signals` | `/getIndexSignalForSkill` | ✅ | Signal catalog/Go |
| `get_signal_combinations` | `/getSignalCombinationForSkill` | ✅ | Signal catalog/Go |
| `get_bot_log_by_type` | `/getBotLogByType` | ✅ | Bot/Go Mongo |
| `generate_grid_strategy` | `/generateGridStrategy` | ⚠️ | Bot → Analyze/Go 规则 |
| `generate_dca_strategy` | `/generateDCAStrategy` | ⚠️ | Bot → Analyze/Go 规则 |
| `loopback_strategy` | `/loopBackStrategy` | ⚠️ | Signal/Go 简化回测 |
| `create_competitor_prompt_template` | `/createCompetitorPromptTemplate` | ✅ | Bot → catalog-api |
| `edit_competitor_prompt_template` | `/editCompetitorPromptTemplate` | ✅ | |
| `delete_competitor_prompt_template` | `/deleteCompetitorPromptTemplate` | ✅ | |
| `create_etf_prompt_template` | `/createEtfPromptTemplate` | ✅ | |
| `edit_etf_prompt_template` | `/editEtfPromptTemplate` | ✅ | |
| `delete_etf_prompt_template` | `/deleteEtfPromptTemplate` | ✅ | |
| `update_pre_market_report` | `/updatePreMarketReport` | ✅ | Bot/Go reports |
| `delete_pre_market_report` | `/deletePreMarketReport` | ✅ | |
| `get_pre_market_reports` | `/getPreMarketReports` | ✅ | |
| `create_intraday_report` | `/createIntradayTradeDecisionReport` | ✅ | |
| `update_intraday_report` | `/updateIntradayTradeDecisionReport` | ✅ | |
| `delete_intraday_report` | `/deleteIntradayTradeDecisionReport` | ✅ | |
| `get_intraday_reports` | `/getIntradayTradeDecisionReports` | ✅ | |
| `create_post_market_report` | `/createPostMarketReport` | ✅ | |
| `update_post_market_report` | `/updatePostMarketReport` | ✅ | |
| `delete_post_market_report` | `/deletePostMarketReport` | ✅ | |
| `get_post_market_reports` | `/getPostMarketReports` | ✅ | |
| **7× Bot/Reminder CRUD×5** | `/create*Bot` … `/get*Log` | ✅ | Bot/Go Mongo；写 Bot ⚠️ 无调度器 |

7 组 Bot 名称前缀：`dca_bot`, `grid_bot`, `smart_trade`, `hdg_bot`, `dca_reminder`, `grid_reminder`, `smart_reminder`（各 `create_` / `update_` / `delete_` / `list_` / `get_*_log`）。

### 3.2 Bespoke 本地工具（21 个）

| Tool | 状态 | 实现 | 默认 chat |
|------|------|------|-----------|
| `search_code` | ✅ | Agent/Go → Signal/Go | ✅ |
| `web_search` | ✅ | Agent/Go DuckDuckGo | ✅ |
| `check_trading_day` | ✅ | Agent/Go → Bot → Data | ✅ |
| `get_current_price` | ✅ | Agent/Go → Bot → Data | ✅ |
| `get_report_bot_codes` | ✅ | Agent/Go → Bot | 🔒 workflow |
| `fetch_market_news` | ⚠️ skipped | Script 未接 | ✅ |
| `fetch_stock_news` | ⚠️ skipped | Script 未接 | ✅ |
| `get_mcp_analysis` | ⚠️ 简化 | Agent/Go → Analyze/Go | ✅ |
| `get_single_prompt_template` | ✅ | Agent/Go → catalog | ✅ |
| `get_capital_flow` | ⚠️ A 股 skip | Agent/Go → Data | ✅ |
| `get_capital_distribution` | ⚠️ A 股 skip | Agent/Go → Data | ✅ |
| `get_bot_yesterday_attitude` | ✅ | Agent/Go → Bot | ✅ / workflow |
| `get_stock_daily_reports` | ✅ | Agent/Go → Bot | ✅ |
| `list_today_reports` | ✅ | Agent/Go → Bot | ✅ |
| `recall_yesterday_summary` | ⚠️ Stub | Agent/Go | 🔒 workflow |
| `read_working_state` | ✅ | Agent/Go WorkingMemory | 🔒 workflow |
| `recall` | ✅ | Agent/Go session 搜索 | ✅ |
| `create_pre_market_report` | ✅ | Agent/Go → Bot | 🔒 workflow |
| `save_local_report` | ✅ | Agent/Go 本地 FS | 🔒 workflow |
| `send_feishu_summary` | ⚠️ Stub | Agent/Go | 🔒 workflow |
| `write_execution_log` | ✅ | Agent/Go 本地 FS | 🔒 workflow |

---

## 四、GeeGooBot 有路由但 Agent 无对应 Tool

| MCP HTTP | 说明 |
|----------|------|
| `POST /getDailyPriceChange` | 日涨跌幅；Agent 未暴露 tool |
| `POST /getUserBotCodes` | 已废弃别名；Agent 用 `get_report_bot_codes` |
| `POST /switchBot` | **不存在**（规划中的 `switch_bot`） |

---

## 五、非 Go 实现说明

| 能力 | 设计实现 | 当前 Agent 状态 |
|------|----------|-----------------|
| 市场/个股新闻 | Python Skill（finance-news / eastmoney-news） | ⚠️ Tool 已注册，runner **未接**，恒 `skipped` |
| 全球免费行情 | Python Skill（global-quotes） | ❌ `fetch_global_quote` 未注册 |
| 旧 Trading LLM 策略/分析 | 原 AIServer 多轮 LLM | **已弃用**；现为 Analyze/Go 规则 + 指标 |
| 富途交易/盘口 | Futu OpenAPI | ⚠️ `trade.NoopProvider`，三接口不可用 |
| 飞书通知 | Webhook HTTP | ⚠️ Stub |

---

## 六、对话场景推荐

| 你想做的事 | 可用 Tool | 避免 |
|------------|-----------|------|
| 查股价 | `search_code` → `get_current_price` | 不要指望 `get_ticker`（未接富途） |
| 技术面分析 | `get_single_prompt_template` → `get_mcp_analysis` | 质量≠旧 LLM；`period` 必填 |
| DCA/网格方案 | `get_signal_combinations` → `generate_dca_strategy` | 确认 analyze-api 已部署；参数要齐 `code`/`name`/`signal_id` |
| 列 Bot | `list_dca_bots` 等 | `switch_bot` 不存在 |
| 新闻 | `web_search` 或外部 | `fetch_*_news` 当前 skipped |
| 持仓 | — | `get_position` 暂不可用 |
| 盘前写报告 | 加 `/toolsets report_workflow` 或跑 workflow | 默认 chat 无 `create_pre_market_report` |

---

## 七、状态汇总矩阵

| 状态 | 数量（约） | 代表 |
|------|------------|------|
| ✅ 完整可用 | **~68** | 大部分 Bot CRUD、报告 CRUD、search、现价、信号列表 |
| ⚠️ 部分/降级 | **~14** | 策略/分析/回测简化、新闻 skipped、富途三接口、stub 三条、A 股资金 |
| ❌ 未注册 | **11** | `switch_bot`, `wait_for_human`, `spawn_subagent`, … |

---

## 八、维护约定

1. 新增 Tool：改 `catalog/catalog.go` 或 `bespoke.go` → 更新 `domains.go` / `toolset.go` → **同步本文件**。  
2. 新增 mcp-api 路由：先改 GeeGooBot `handler.go` + `implemented-routes.md`。  
3. 禁止把本文件当自动生成物；以代码为准，本文是**可读树形视图**。  
4. 相关文档：[interface-map.md](./geegoo-mcp/interface-map.md) · [tool-catalog.md](../architecture/layers/L2-tools/tool-catalog.md)

---

## 九、规划未注册（仅文档 / Skill 提及）

| Tool | Phase | 说明 |
|------|-------|------|
| `switch_bot` | 6 | 启停 Bot；domains 已列，Registry 与 mcp-api **均未实现** |
| `wait_for_human` | 6 | Bot 创建前人工确认 |
| `spawn_subagent` | 2+ | StockAnalyst / NewsCollector 子 Agent |
| `fetch_global_quote` | 5 | global-quotes 脚本 |
| `get_tech_prompt_list` | 4 | 遗留；用 `get_single_prompt_template` |
| `recall_past_attitude` | 2 | attitude_history.jsonl |
| `recall_similar_setup` | 4+ | 向量检索 |
| `compare_daily_reports` | 3 | 报告差分 |
| `get_daily_reports_unified` | 4 | reportServer :6100 |
| `update_working_state` | 2 | Runtime 内部 |
| `emit_event` | 0 | 调试 EventBus |

---

*最后核对：`go test ./internal/tools/... -run TestRegisterAllToolCount` 要求 registered ≥ 80。*
