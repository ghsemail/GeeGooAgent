# Skill 与 Tool 分类

> **SSOT**：[interface-map.md](../../reference/geegoo-mcp/interface-map.md)  
> **背景**：原 `geegoo` + `geegoo_agent` 双 Skill 已合并为 **`~/.cursor/skills/geegoo`**，统一 geegoo mcp :5700。  
> GeeGoo Agent 运行时：**定时报告走 Skill**，**可复用能力走 Tool**。

## 1. 一句话原则

| 层级 | 是什么 | 判断标准 |
|------|--------|----------|
| **Skill（L5）** | 有固定 **workflow + 报告模板 + supervisor** 的**业务场景** | 要定时/触发跑完一整条链路，产出一份「某类报告」 |
| **Tool（L2）** | 对 GeeGoo HTTP / 本地脚本的**原子封装** | 单步可调用、可被多个 Skill 或对话复用 |
| **Bundled** | Skill 内部的**脚本依赖** | 不单独暴露给 LLM；由 Tool 在内部 subprocess 调用 |
| **LLM Task** | 窄域结构化推理 | 非 Tool Registry 条目；由 workflow 在固定步骤调用 |

```text
geegoo-mcp 文档领域（12）     →  GeeGoo Agent 归类
─────────────────────────────────────────────
reports / trading           →  pre_market · intraday · post_market Skill
analyst                       →  Tool 池（分析原子）
common / trading              →  Tool 池（账户 · 行情 · 搜码）
strategy                      →  Tool 池（策略 Skill）
bot / reminder ×6             →  Tool 池（bot_manager）
本地新闻脚本                  →  Bundled（finance-news 等）
```

---

## 2. Skill 全景（L5）

每个 Skill = `skills/<name>/` 目录 + `manifest.yaml` 白名单 + workflow 步骤表。

| Skill | TradingBot 依据 | 触发 | Phase | 状态 |
|-------|-----------------|------|-------|------|
| **`pre_market`** | `geegoo-mcp/market/reports.md` §盘前 | timer 08:00 | 1 | ✅ 已实现 |
| **`intraday`** | `geegoo-mcp/market/reports.md` §盘中交易决策 | webhook / 信号 | 3 | 📋 规划 |
| **`post_market`** | `geegoo-mcp/market/reports.md` §盘后 | timer 17:00 | 2 | 📋 规划 |
| **`on_demand_analysis`** | `geegoo-mcp/analyst/agent-analyst.md` | chat 按需 | 4 | 📋 规划 |
| **`strategy`** | `geegoo-mcp/strategy/` | chat 按需 | 5 | 📋 规划 |
| **`bot_manager`** | `geegoo-mcp/bot/` · `reminder/` | chat + `wait_for_human` | 6 | 📋 规划 |

### 2.1 三个「报告 Skill」的边界（Market 域）

三者共用部分 Tool，但 **workflow、模板、supervisor、入库 API 不同**：

| | 盘前 `pre_market` | 盘中 `intraday` | 盘后 `post_market` |
|--|-------------------|-----------------|---------------------|
| **目的** | 开盘前综合预判 | 交易决策 / 执行摘要 | 收盘复盘 |
| **核心写库** | `createPreMarketReport` | `createIntradayTradeDecisionReport` | `createPostMarketReport` |
| **bot_id** | 可选（来自监控列表） | **必填** | 可选 |
| **典型独有数据** | 指数 hourly、市场新闻、周线 | 实时价、持仓、信号 | vs_pre_market 对照 |
| **Phase 1 Tool 子集** | 见 `skills/pre_market/manifest.yaml` | 待定义 | 待定义 |

### 2.2 非报告类 Skill（对话/按需）

| Skill | 做什么 | 主要用哪些 Tool |
|-------|--------|-----------------|
| `on_demand_analysis` | 用户问「分析一下 XXX」 | `search_code`, `get_mcp_analysis`, `get_single_prompt_template`, `fetch_stock_news` |
| `strategy` | 网格/DCA 参数建议、回测 | `get_index_signals`, `generate_*_strategy`, `loopback_strategy` |
| `bot_manager` | 创建/改 Bot、提醒 | 全部 `create_*_bot/reminder`, `search_code`, `get_position` |

---

## 3. Tool 池（L2）— 按 TradingBot 文档分域

Tool **不属于某个 Skill 独占**；Skill 通过 `manifest.yaml` 的 `tools:` 白名单订阅子集。

### 3.1 reports / trading — 主要服务「报告 Skill」

来源：[reports.md](../../reference/geegoo-mcp/market/reports.md)、[trading-data.md](../../reference/geegoo-mcp/market/trading-data.md)

| Tool | API | 典型消费者 |
|------|-----|------------|
| `check_trading_day` | `/checkTradingDay` | 三个报告 Skill 前置 |
| `get_report_bot_codes` | `/getReportBotCodes` | pre_market / post_market |
| `get_capital_flow` | `/getCapitalFlow` | pre_market, post_market |
| `get_capital_distribution` | `/getCapitalDistribution` | pre_market, post_market |
| `get_bot_yesterday_attitude` | `/getBotYesterdayAttitude` | pre_market |
| `create_pre_market_report` | `/createPreMarketReport` | **pre_market** |
| `create_intraday_trade_decision_report` | `/createIntradayTradeDecisionReport` | **intraday**（未实现） |
| `create_post_market_report` | `/createPostMarketReport` | **post_market**（未实现） |
| `get_pre_market_reports` / update / delete | 5700 | 查询维护（Tool，非 Skill） |

### 3.2 AgentAnalyst 分析原子 — **Tool，不是 Skill**

来源：[agent-analyst.md](../../reference/geegoo-mcp/analyst/agent-analyst.md)

| Tool | 说明 | 消费者 |
|------|------|--------|
| `get_mcp_analysis` | 技术面/指数 LLM 分析 | 三个报告 Skill + on_demand |
| `get_single_prompt_template` | 拉 prompt 列表 | on_demand |
| `get_tech_prompt_list` | 遗留别名 | on_demand |
| `create/edit/delete_*_prompt_template` | 用户自建模板 | on_demand（高级） |
| `get_stock_daily_reports` | 按日聚合三类报告 | 所有报告 Skill（幂等/对照） |
| `list_today_reports` | 幂等别名 | 报告 Skill |

### 3.3 Common 通用（5700）— **Tool**

来源：`geegoo-mcp/common.md`

| Tool | 说明 | 消费者 |
|------|------|--------|
| `search_code` | 标的搜索 | bot_manager、on_demand、策略前必调 |
| `get_position` | 富途持仓 | intraday、SmartTrade |
| `get_current_price` | 最新价 | intraday、on_demand |
| `get_index_signals` | 指标信号列表 | strategy |
| `get_signal_combinations` | 组合信号 | strategy |

### 3.4 Strategy（生成 + 回测）— **Tool**

来源：[strategy/README.md](../../reference/geegoo-mcp/strategy/README.md)

| Tool | 说明 |
|------|------|
| `generate_grid_strategy` | 网格参数建议 |
| `generate_dca_strategy` | DCA 参数建议 |
| `loopback_strategy` | 回测 |

### 3.5 Bot / Reminder CRUD — **Tool**（仅 `bot_manager` Skill）

来源：`geegoo-mcp/bot/*.md`、`geegoo-mcp/reminder/*.md`（见 [interface-map.md](../../reference/geegoo-mcp/interface-map.md)）

| 组 | Tool 模式 |
|----|-----------|
| DCA/GRID/SmartTrade/HDG Bot | `create/update/delete/list_*_bot`, `get_*_bot_log` |
| DCA/GRID/Smart Reminder | `create/update/delete/list_*_reminder`, `get_*_reminder_log` |

Phase 1 **scheduled Skill 不得暴露**上述 CRUD（见 `skills.md` Tool 过滤规则）。

### 3.6 本地 / 横切 — **Tool 或基础设施**

| Tool | 类型 | 说明 |
|------|------|------|
| `fetch_market_news` | Tool → Bundled | US/CN/HK 市场新闻 |
| `fetch_stock_news` | Tool → Bundled | 个股新闻 |
| `fetch_global_quote` | Tool → Bundled | 免费行情（Phase 5） |
| `save_local_report` | Tool | 本地 md 留档 |
| `send_feishu_summary` | Tool | 推送摘要 |
| `write_execution_log` | Tool | 横切日志 |
| `read_working_state` | Tool | 读 WorkingMemory |
| `recall_yesterday_summary` | Tool | Episodic 摘要 |

### 3.7 LLM Task（非 Tool）

| 任务 | 归属 Skill | 状态 |
|------|------------|------|
| `parse_weekly_analysis` | pre_market | ✅ |
| `synthesize_pre_market_report` | pre_market | ✅ |
| `parse_market_sentiment` | pre_market | 📋 |
| （盘中/盘后合成） | intraday / post_market | 📋 |

---

## 4. 原「两个 Skill」如何映射到现结构

| 原 Cursor Skill | 现 GeeGoo Agent |
|-----------------|---------------|
| **geegoo** 盘前 cron | → **`pre_market` Skill** + 一组 Market/News Tool |
| **geegoo** 盘中/盘后（若有） | → **`intraday` / `post_market` Skill**（待建） |
| **geegoo** 技术分析问答 | → **`on_demand_analysis` Skill** + AgentAnalyst **Tool** |
| **geegoo** 策略/回测 | → **`strategy` Skill** + Strategy **Tool** |
| **geegoo** Bot CRUD | → **`bot_manager` Skill** + Bot **Tool** |
| **geegoo** / **geegoo** 内嵌新闻脚本 | → **Bundled**（不变） |

**合并的是运行时与 Client 层**，不是把「策略/Bot/盘前」塞进同一个 Skill。

---

## 5. 当前仓库实况（Phase 1）

```
skills/
├── pre_market/          ← 唯一完整 Skill
├── bundled/             ← 新闻脚本（非 Skill）
│   ├── finance-news/
│   ├── eastmoney-news/
│   └── free-stock-global-quotes-news/
└── (intraday, post_market, … 待建)

src/geegoo/tools/    ← 16 个 Tool 全部注册，manifest 白名单过滤
```

| 已注册 Tool | 在 pre_market workflow 中使用 |
|-------------|------------------------------|
| 14 个 | ✅ |
| `recall_yesterday_summary` | ❌ 仅注册 |
| `read_working_state` | ❌ 仅注册 |
| `send_feishu_summary` | ❌ 可选未默认接入 |
| `get_stock_daily_reports` | ⚠️ 由 `list_today_reports` 间接使用 |

---

## 6. 推荐目录演进（消除「乱」）

1. **新增 Skill 壳**：`skills/intraday/`、`skills/post_market/`（manifest + workflow 占位 + supervisor 占位）。
2. **manifest 只列本 Skill 需要的 Tool**，不要复制 16 个全集。
3. **tool-catalog.md** 表头增加列：`TradingBot 文档` | `默认归属 Skill` | `实现状态`。
4. **bundled 保持不在 Skill 列表**，仅在 manifest `bundled:` 声明依赖。

---

## 7. 快速查阅：我该加 Skill 还是 Tool？

```text
问：用户要在 8:00 自动出一份盘前报告？
答：Skill（pre_market）+ 若干 Tool

问：用户聊天说「帮我用周线模板分析腾讯」？
答：on_demand Skill（或临时 workflow）+ get_mcp_analysis Tool

问：新增「查询资金分布」能力？
答：只加 Tool；被 pre_market / post_market 引用

问：新增「创建 DCA 机器人」？
答：Tool + bot_manager Skill（交互式）；scheduled Skill 禁用
```
