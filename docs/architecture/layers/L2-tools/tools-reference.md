# Tools 完整参考（82）

> **你要找的「每个 Tool 干什么、从哪调、实现没实现」看这一篇。**  
> 代码 SSOT：`internal/tools/catalog/catalog.go` · `bespoke.go` · `httpbackend.go`  
> MCP 路由 SSOT：[interface-map.md](../../../reference/geegoo-mcp/interface-map.md)

## 图例

| 状态 | 含义 |
|------|------|
| ✅ | 已注册，正常环境端到端可用 |
| ⚠️ | 已注册，但降级 / 依赖未配 / 后端简化 |
| ❌ | Agent 侧未实现（调用即 `skipped`） |

**类型**：`http` = 通用 HTTP 转发；`bespoke` = 手写 Handler（可含本地逻辑）。

**服务器缩写**：`Bot:3120` GeeGooBot mcp-api · `Sig:3200` signal-api · `Cat:3210` catalog-api · `Local` 本机 · `Web` 外网

---

## 未实现 / 有问题（优先看）

| Tool | 状态 | 作用 | 来源 | 接口 | 问题 |
|------|------|------|------|------|------|
| `fetch_market_news` | ✅ | 市场新闻 US/CN/HK | Local 脚本 | subprocess `skills/bundled/finance-news` | 无 Python/脚本时 skip |
| `fetch_stock_news` | ✅ | 个股新闻 | Local 脚本 | subprocess | 同上 |
| `recall_yesterday_summary` | ✅ | 昨日盘前摘要 | Local 文件 | 读 `reports/{date}/{code}-premarket.md` | 无昨日文件时 skip；可回退 MCP |
| `get_ticker` | ⚠️ | 逐笔行情 | Bot:3120 | `POST /getTicker` | 富途/MCP 未配时常空 |
| `get_broker` | ⚠️ | 经纪席位分布 | Bot:3120 | `POST /getBroker` | 同上 |
| `get_position` | ⚠️ | 账户持仓 | Bot:3120 | `POST /getPosition` | 富途未配；空 data → skip |
| `get_capital_flow` | ⚠️ | 资金流向 | Bot:3120 | `POST /getCapitalFlow` | 依赖 MCP 数据源；空结果可能 skip |
| `get_capital_distribution` | ⚠️ | 资金分布 | Bot:3120 | `POST /getCapitalDistribution` | 同上 |
| `get_mcp_analysis` | ⚠️ | 技术面/指数分析 | Analyze:3230 → Bot:3120 | `POST /getMCPAnalysis` | 优先 analyze-api；质量取决于后端 |
| `generate_grid_strategy` | ⚠️ | 网格参数建议 | Analyze:3230 | `POST /generateGridStrategy` | analyze-api 未部署时回退 Bot:3120 |
| `generate_dca_strategy` | ⚠️ | DCA 参数建议 | Analyze:3230 | `POST /generateDCAStrategy` | 同上 |
| `loopback_strategy` | ⚠️ | 策略回测 | Sig:3200 | `POST /loopBackStrategy` | Signal/Go **简化**确定性回测 |
| `send_feishu_summary` | ⚠️ | 飞书推送摘要 | Web | webhook URL | 已实现 POST；未配 `feishu_webhook_url` → skip |

**说明**：Registry 中 **82 个 Tool 均已注册**，没有「未注册」的幽灵 Tool。上表是**运行态不可用或降级**的项；其余默认可用（仍受 MCP 鉴权、网络、参数约束）。

---

## 一、感知 Perception

| Tool | 作用 | 类型 | 来源 | HTTP / 实现 | 状态 | 备注 |
|------|------|------|------|-------------|------|------|
| `check_trading_day` | 是否交易日 | bespoke | Bot:3120 | `POST /checkTradingDay` | ✅ | |
| `search_code` | 标的搜索 | bespoke | Sig:3200 | `POST /searchCode` | ✅ | 直连 signal-api |
| `web_search` | 网页搜索 | bespoke | Web | DuckDuckGo HTML | ✅ | 非 MCP |
| `get_current_price` | 最新价快照 | bespoke | Bot:3120 | `POST /getCurrentPrice` | ✅ | 失败可回退 `/getTicker` |
| `get_ticker` | 逐笔行情 | http | Bot:3120 | `POST /getTicker` | ⚠️ | 见上表 |
| `get_broker` | 经纪分布 | http | Bot:3120 | `POST /getBroker` | ⚠️ | |
| `get_position` | 持仓 | http | Bot:3120 | `POST /getPosition` | ⚠️ | mcp_token |
| `get_report_bot_codes` | 报告待分析标的 | bespoke | Bot:3120 | `POST /getReportBotCodes` | ✅ | workflow |
| `fetch_market_news` | 市场新闻 | bespoke | Local | 脚本 | ❌ | 见上表 |
| `fetch_stock_news` | 个股新闻 | bespoke | Local | 脚本 | ❌ | 见上表 |

---

## 二、分析 Analysis

| Tool | 作用 | 类型 | 来源 | HTTP / 实现 | 状态 | 备注 |
|------|------|------|------|-------------|------|------|
| `get_mcp_analysis` | MCP 技术分析 | bespoke | Bot:3120 | `POST /getMCPAnalysis` | ⚠️ | `period` 必填 |
| `get_single_prompt_template` | Prompt 模板列表 | bespoke | Bot:3120 | `POST /getSinglePromptTemplate` | ✅ | type: index/tech/fundamental |
| `get_capital_flow` | 资金流向 | bespoke | Bot:3120 | `POST /getCapitalFlow` | ⚠️ | A 股 skip |
| `get_capital_distribution` | 资金分布 T-1 | bespoke | Bot:3120 | `POST /getCapitalDistribution` | ⚠️ | A 股 skip |
| `get_bot_yesterday_attitude` | 昨日态度 | bespoke | Bot:3120 | `POST /getBotYesterdayAttitude` | ✅ | 404→neutral |
| `get_bot_log_by_type` | Bot 运行日志 | http | Bot:3120 | `POST /getBotLogByType` | ✅ | |
| `get_stock_daily_reports` | 按日聚合报告 | bespoke | Bot:3120 | `POST /getStockDailyReports` | ✅ | |
| `list_today_reports` | 同日幂等检查 | bespoke | Bot:3120 | `POST /getStockDailyReports` | ✅ | 别名 |
| `get_index_signals` | 指标信号列表 | http | Cat:3210 | `POST /getIndexSignalForSkill` | ✅ | |
| `get_signal_combinations` | 组合信号 | http | Cat:3210 | `POST /getSignalCombinationForSkill` | ✅ | |
| `generate_grid_strategy` | 网格策略建议 | http | Bot:3120 | `POST /generateGridStrategy` | ⚠️ | |
| `generate_dca_strategy` | DCA 策略建议 | http | Bot:3120 | `POST /generateDCAStrategy` | ⚠️ | |
| `loopback_strategy` | 策略回测 | http | Sig:3200 | `POST /loopBackStrategy` | ⚠️ | 直连 signal-api |
| `create_competitor_prompt_template` | 建竞品模板 | http | Bot:3120 | `POST /createCompetitorPromptTemplate` | ✅ | Phase 7 |
| `edit_competitor_prompt_template` | 改竞品模板 | http | Bot:3120 | `POST /editCompetitorPromptTemplate` | ✅ | |
| `delete_competitor_prompt_template` | 删竞品模板 | http | Bot:3120 | `POST /deleteCompetitorPromptTemplate` | ✅ | |
| `create_etf_prompt_template` | 建 ETF 模板 | http | Bot:3120 | `POST /createEtfPromptTemplate` | ✅ | |
| `edit_etf_prompt_template` | 改 ETF 模板 | http | Bot:3120 | `POST /editEtfPromptTemplate` | ✅ | |
| `delete_etf_prompt_template` | 删 ETF 模板 | http | Bot:3120 | `POST /deleteEtfPromptTemplate` | ✅ | |

### Bot / Reminder 日志（各 1×）

| Tool | 作用 | 来源 | HTTP | 状态 |
|------|------|------|------|------|
| `get_dca_bot_log` | DCA Bot 日志 | Bot:3120 | `POST /getDCABotLog` | ✅ |
| `get_grid_bot_log` | GRID Bot 日志 | Bot:3120 | `POST /getGRIDBotLog` | ✅ |
| `get_smart_trade_log` | SmartTrade 日志 | Bot:3120 | `POST /getSmartTradeLog` | ✅ |
| `get_hdg_bot_log` | HDG Bot 日志 | Bot:3120 | `POST /getHDGBotLog` | ✅ |
| `get_dca_reminder_log` | DCA 提醒日志 | Bot:3120 | `POST /getDCAReminderLog` | ✅ |
| `get_grid_reminder_log` | GRID 提醒日志 | Bot:3120 | `POST /getGRIDReminderLog` | ✅ |
| `get_smart_reminder_log` | Smart 提醒日志 | Bot:3120 | `POST /getSmartReminderLog` | ✅ |

---

## 三、决策 Decision

| Tool | 作用 | 类型 | 来源 | 实现 | 状态 | 备注 |
|------|------|------|------|------|------|------|
| `recall` | 跨会话检索历史 | bespoke | Local | SQLite FTS | ✅ | chat |
| `recall_yesterday_summary` | 昨日报告摘要 | bespoke | Local | 未写 | ❌ | 见上表 |
| `read_working_state` | 读 workflow 进度 | bespoke | Local | working_state | ✅ | 一般不暴露 chat |

---

## 四、行动 Action — 报告

| Tool | 作用 | 类型 | 来源 | HTTP | 状态 |
|------|------|------|------|------|------|
| `create_pre_market_report` | 写盘前报告 | bespoke | Bot:3120 | `POST /createPreMarketReport` | ✅ |
| `update_pre_market_report` | 改盘前报告 | http | Bot:3120 | `POST /updatePreMarketReport` | ✅ |
| `delete_pre_market_report` | 删盘前报告 | http | Bot:3120 | `POST /deletePreMarketReport` | ✅ |
| `get_pre_market_reports` | 查盘前报告 | http | Bot:3120 | `POST /getPreMarketReports` | ✅ |
| `create_intraday_report` | 写盘中报告 | http | Bot:3120 | `POST /createIntradayTradeDecisionReport` | ✅ |
| `update_intraday_report` | 改盘中报告 | http | Bot:3120 | `POST /updateIntradayTradeDecisionReport` | ✅ |
| `delete_intraday_report` | 删盘中报告 | http | Bot:3120 | `POST /deleteIntradayTradeDecisionReport` | ✅ |
| `get_intraday_reports` | 查盘中报告 | http | Bot:3120 | `POST /getIntradayTradeDecisionReports` | ✅ |
| `create_post_market_report` | 写盘后报告 | http | Bot:3120 | `POST /createPostMarketReport` | ✅ |
| `update_post_market_report` | 改盘后报告 | http | Bot:3120 | `POST /updatePostMarketReport` | ✅ |
| `delete_post_market_report` | 删盘后报告 | http | Bot:3120 | `POST /deletePostMarketReport` | ✅ |
| `get_post_market_reports` | 查盘后报告 | http | Bot:3120 | `POST /getPostMarketReports` | ✅ |
| `save_local_report` | 本地 md 留档 | bespoke | Local | 写 workspace | ✅ |
| `send_feishu_summary` | 飞书摘要 | bespoke | Web | webhook | ⚠️ |

---

## 五、行动 Action — 交易 Bot（×4）

每组 5 个 Tool：`create_*` `update_*` `delete_*` `list_*` `get_*_log`。来源均为 **Bot:3120**，状态 **✅**（写操作 chat 需 ApprovalGate）。

| 前缀 | 创建 HTTP | 列表 HTTP |
|------|-----------|-----------|
| `dca_bot` | `POST /createDCABot` | `POST /getAllDCABots` |
| `grid_bot` | `POST /createGRIDBot` | `POST /getAllGRIDBots` |
| `smart_trade` | `POST /createSmartTrade` | `POST /getAllSmartTrades` |
| `hdg_bot` | `POST /createHDGBot` | `POST /getAllHDGBots` |

update/delete/log 路径见 [interface-map.md](../../../reference/geegoo-mcp/interface-map.md) 对应 bot 域。

**问题**：GeeGooBot 侧 **无 Bot 自动交易 scheduler**；Agent 只 CRUD 配置。

---

## 六、行动 Action — 提醒 Bot（×3）

同上，来源 **Bot:3120**，状态 **✅**。

| 前缀 | 创建 HTTP | 列表 HTTP |
|------|-----------|-----------|
| `dca_reminder` | `POST /createDCAReminder` | `POST /getAllDCAReminders` |
| `grid_reminder` | `POST /createGRIDReminder` | `POST /getAllGRIDReminders` |
| `smart_reminder` | `POST /createSmartReminder` | `POST /getAllSmartReminders` |

---

## 七、Meta

| Tool | 作用 | 类型 | 来源 | 状态 |
|------|------|------|------|------|
| `write_execution_log` | workflow 步骤日志 | bespoke | Local | ✅ |

---

## 默认 chat 能否调用？

| Toolset | 数量 | 默认 chat |
|---------|------|-----------|
| `market` | 17 | ✅ |
| `strategy` | 3 | ✅ |
| `bot_manager` | 20 | ✅ |
| `reminder_manager` | 15 | ✅ |
| `report_query` | 10 | ✅ |
| `report_workflow` | 8 | 🔒 需 `/toolsets report_workflow` 或 `geegoo run` |

详见 [toolsets.md](./toolsets.md)、[tools-tree.md](./tools-tree.md)。

---

## 相关文档分工

| 文档 | 用途 |
|------|------|
| **本文件** | 全量 82 Tool：作用、来源、接口、状态、问题 |
| [tools-tree.md](./tools-tree.md) | 树形总览、踩坑、chat 场景 |
| [tool-server-mapping.md](./tool-server-mapping.md) | 生产 IP、间接依赖、场景子集 |
| [tool-catalog.md](./tool-catalog.md) | Phase/MVP、参数校验规则 |
| [interface-map.md](../../../reference/geegoo-mcp/interface-map.md) | MCP 73 路由与 geegoo Skill 对照 |

维护：改 Tool 后同步 **本文件** + `catalog.go` + `interface-map`（若新 HTTP）。
