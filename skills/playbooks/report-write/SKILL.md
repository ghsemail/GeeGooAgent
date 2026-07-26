---
name: report-write
description: 写入报告、保存报告、创建盘中报告、创建盘后报告、修改报告、补写报告、落库报告、生成报告正文后提交。用户说「把分析写成报告」「保存盘前/盘后」「更新报告内容」时触发（盘前新建走 workflow，不在此）。
---

# 报告写入（Chat）

## 适用 Toolset

`report_write` · `analyst_runtime` · `market`（生成正文时常用）

## 与查询 / 自动化的边界

| 用户意图 | 用 |
|----------|-----|
| 查已有报告 | `report-lookup` → `report_query` |
| Chat 写入/修改报告正文 | **本 playbook** → `report_write` |
| 自动生成完整盘前流水线 | `geegoo run pre_market`（**不要**在 chat 调 `create_pre_market_report`） |

## 可写范围

| 类型 | create | update | delete |
|------|--------|--------|--------|
| 盘中 | `create_intraday_report` ✅ | `update_intraday_report` | `delete_intraday_report` |
| 盘后 | `create_post_market_report` ✅ | `update_post_market_report` | `delete_post_market_report` |
| 盘前 | ❌ Chat 无 create | `update_pre_market_report` | `delete_pre_market_report` |

## 标准流程（新建盘中/盘后）

1. **确认标的** — `search_code` → `code`、`stock_name`
2. **（推荐）生成正文** — 若用户尚未给正文：
   - `get_single_prompt_template`（`type=tech`，选 `period`）→ `get_mcp_analysis`
   - 可选：`fetch_stock_news`、`get_capital_flow`
   - 将分析整理为 Markdown（符合 SOUL：## 标题、列表，无宽表格）
3. **向用户展示摘要** — 确认类型（盘中/盘后）、日期、正文要点
4. **用户批准后写入** — `create_intraday_report` 或 `create_post_market_report`：
   - `code`、`stock_name` 必填
   - `report_date` 可选，默认今天（`YYYY-MM-DD`）
   - `content` = 报告正文 markdown
5. **验证** — `get_*_reports` 或 `get_stock_daily_reports` 确认已入库

## 标准流程（修改已有）

1. `get_pre_market_reports` / `get_intraday_reports` / `get_post_market_reports`（或 `get_stock_daily_reports`）定位 **`report_id`**
2. 展示变更 diff 要点 → 用户确认
3. `update_*_report`（`report_id` + `content`）

## 硬规则

- 所有 create/update/delete 须 **用户明确批准**
- **禁止**在 interactive chat 调用 `create_pre_market_report`（会被拦截；盘前新建仅 workflow）
- 写入前尽量先查是否已有同日同标的报告，避免重复（`list_today_*` / `get_*_reports`）
- `content` 须完整 markdown，不要只写一句话占位

## 反模式

- 未生成/未确认正文就 create
- 把「帮我跑盘前自动化」当成 create_pre_market_report
- 用 `save_local_report` 代替 API 落库（`save_local_report` 仅 workflow 本地留档）

## 输出

成功后返回 `report_id`、类型、日期、标的；失败说明 API 原因与已尝试步骤。
