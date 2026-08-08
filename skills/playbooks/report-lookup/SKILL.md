---
name: report-lookup
description: 盘前报告、盘后报告、盘中报告、今日报告、报告查询、Bot 昨日态度、get_stock_daily_reports、复盘。用户问「今天盘前写了什么」「查报告」「昨日态度」时触发。
---

# 报告查询与对照

## 适用 Toolset

`report_query` · `market`（可选 `search_code` 收窄标的）

## 标准流程

1. **按日聚合** — `get_stock_daily_reports`（建议 `report_date`=`YYYY-MM-DD`，可选 `code`）
2. **今日盘前是否已生成** — `list_today_reports`（幂等检查）
3. **今日盘后** — `list_today_stock_postmarket_reports`
4. **单类列表** — `get_stock_premarket_reports` / `get_intraday_reports` / `get_stock_postmarket_reports`
5. **Bot 监控结论** — `get_bot_yesterday_attitude`（需 `bot_id`；先 `list_*_bots` 定位）

## 与自动化 Workflow 的边界

| 用户意图 | 用 |
|----------|-----|
| 「查已有报告」 | 本 playbook（report_query） |
| 「帮我跑一遍盘前自动化」 | `geegoo run premarket_market` / scheduler（**不要**在 chat 里乱调 `create_stock_premarket_report`） |
| Chat 补写/修改报告正文 | `report_write` toolset（`update_*_report` 等） |

## 硬规则

- 无报告时说明日期/标的，建议换日期或确认是否交易日（`check_trading_day`）
- 引用报告内容注明类型（盘前/盘中/盘后）与日期
- 查 Bot 态度必须先有 **bot_id**

## 反模式

- 用 `get_report_bot_codes` 查用户个人 Bot（那是 report workflow 监控列表）
- 把 workflow 专用 `recall_yesterday_summary` 当唯一数据源（优先 API 报告）

## 输出

按时间线摘要；多报告时用小标题区分；缺数据列明已尝试的 Tool。
