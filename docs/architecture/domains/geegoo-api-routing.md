# geegoo mcp API 路由（5700）

> **SSOT**：[TradingBot docs/geegoo-mcp/interface-map.md](D:/Geegoo/TradingBot/docs/geegoo-mcp/interface-map.md)

原 geegoo mcp（5700）已合并入 **geegoo mcp :5700**。GeeGoo Agent 单一 `GeeGooBotClient` / `MarketClient` 别名。

## 认证

| 层 | 说明 |
|----|------|
| Header | `Authorization: Bearer <sk-...>` |
| Body | `mcp_token`（需用户身份的接口必填） |

## Workflow 常用接口

| 需求 | HTTP | GeeGoo Agent Tool |
|------|------|-----------------|
| 交易日 | `/checkTradingDay` | `check_trading_day` |
| 报告待分析标的 | `/getReportBotCodes` | `get_report_bot_codes` |
| 资金流向 / 分布 | `/getCapitalFlow` · `/getCapitalDistribution` | `get_capital_flow` · `get_capital_distribution` |
| 昨日态度 | `/getBotYesterdayAttitude` | `get_bot_yesterday_attitude` |
| Bot 运行日志 | `/getBotLogByType` | `get_bot_log_by_type` |
| 技术分析 | `/getMCPAnalysis` | `get_mcp_analysis` |
| 按日聚合报告 | `/getStockDailyReports` | `get_stock_daily_reports` |
| 创建盘前报告 | `/createPreMarketReport` | `create_pre_market_report` |

## 报告查询优先级

| 场景 | 首选 |
|------|------|
| 按日期聚合 pre/intraday/post | `getStockDailyReports` |
| 按 code 查盘前报告列表 | `getPreMarketReports` |
| 盘后本地缺失兜底 | `getPreMarketReports` |

## 文档领域（12）

见 [architecture.md](../../../reference/geegoo-mcp/architecture.md)：`common` · `trading` · `reports` · `analyst` · `strategy` · bot×4 · reminder×3

## 参考

- geegoo Skill：`~/.cursor/skills/geegoo/docs/geegoo-mcp/`
- Cursor 路由：`~/.cursor/skills/geegoo/ROUTING.md`
