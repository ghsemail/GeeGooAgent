# geegoo Skill → GeeGoo Agent 映射

对照 Cursor Skill `geegoo`（`~/.cursor/skills/geegoo/SKILL.md` 与 `geegoo/docs/`）。HTTP 总表见 [interface-map.md](../../reference/geegoo-mcp/interface-map.md)。

## SKILL 章节 → Tool

| SKILL 章节 | MCP 接口 | Agent Tool | 状态 |
|------------|----------|------------|------|
| 股票技术面分析 | getSinglePromptTemplate, getMCPAnalysis | get_single_prompt_template, get_mcp_analysis | ✅ / ⚠️ 简化 |
| 股票日常报告查询 | getStockDailyReports | get_stock_daily_reports, list_today_reports | ✅ |
| 信号指标 | getIndexSignalForSkill, getSignalCombinationForSkill | get_index_signals, get_signal_combinations | ✅ |
| 策略生成 | generateGridStrategy, generateDCAStrategy | generate_grid_strategy, generate_dca_strategy | ⚠️ |
| 策略回测 | loopBackStrategy | loopback_strategy | ⚠️ 简化 |
| DCA/GRID/Smart/HDG Bot | create/update/delete/getAll + log | create/update/delete/list_* + get_*_log | ✅ |
| DCA/GRID/Smart Reminder | 同上 | idem *_reminder | ✅ |
| common / trading | searchCode, getPosition, getCurrentPrice, getReportBotCodes | search_code, get_position, get_current_price, get_report_bot_codes | ✅ / ⚠️ 富途 |
| AgentAnalyst 竞品/ETF | Competitor/Etf PromptTemplate CRUD | create/edit/delete_*_prompt_template | ⚠️ |

## 端口

| 端口 | 用途 |
|------|------|
| **3120** | GeeGooBot mcp-api（默认全部 HTTP Tool） |
| **3200** | signal-api（search_code, loopback_strategy） |
| **3210** | catalog-api（指标、组合信号） |

## 交互规范

1. 任何 `create_*_bot` / `create_*_reminder` 前必须 `search_code`
2. Chat 写操作经 **ApprovalGate** 确认
3. GRID Reminder 显式 `frequency: 60m`
4. SmartTrade sell_only：创建前 `get_position`，不传 price

## 资产与代码对齐（pre_market）

| 原 geegoo 资产 | GeeGooAgent 现状 |
|----------------|------------------|
| pre-market workflow / template | `skills/pre_market/` + `internal/workflow/premarket.go` |
| post-market / intraday 参考 | **未迁移**；Skill 名已注册，步骤为空 |
| finance-news 等脚本 | `skills/bundled/`（Tool 侧 script runner **未实现**） |
| cron JSON | **废弃** → `geegoo scheduler` + `jobs.json` |
| geegoo-mcp 文档镜像 | `docs/reference/geegoo-mcp/` + `domains/` |

完整 Tool 表 → [layers/L2-tools/tool-catalog.md](../layers/L2-tools/tool-catalog.md)
