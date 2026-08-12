---
name: stock-analysis
description: 个股技术面分析、指数分析、prompt_id、get_mcp_analysis、周线日线、模板列表、资金面、个股新闻。用户问「分析一下」「技术面」「用什么模板」「EMA/MACD 分析」时触发。
---

# 个股 / 指数分析（运行时）

## 适用 Toolset

`market` + `analyst_runtime`（默认 chat 已包含）

## 标准流程

1. **确认标的** — `search_code`（名称/代码模糊搜；特殊标的如 SpaceX 也在库内）
2. **选分析周期** — 未说明时默认 `daily`；「这周」→ `weekly`；可 `clarify` 确认
3. **选 Prompt 模板** — `get_single_prompt_template`（技术面 **必传 tag**）：
   | 用户意图 | 调用 |
   |----------|------|
   | 股价 / 价格 / 涨跌 | `type=tech`, `tag=price` |
   | K线 / 形态 | `type=tech`, `tag=kline` |
   | 趋势 / 走势 | `type=tech`, `tag=flag` |
   | 资金 / 主力 | `type=tech`, `tag=capital_flow` |
   | MACD / EMA 等 | `type=index` 或 `get_single_prompt_template_by_index` |
   | 财报 / 估值 | `type=fundamental` |
   - 有 **`selected_prompt_id`** → 直接用于下一步；勿无 tag 拉全量 tech
   - 已知指标名：`get_single_prompt_template_by_index`（`period` 必填）
4. **执行分析** — `get_mcp_analysis`（`name`、`code`、`prompt_id`、`period`；`language` 默认 `cn`）
5. **可选增强**（用户关心资金/新闻时）
   - `get_capital_flow` / `get_capital_distribution`
   - `fetch_stock_news`

## 硬规则

- 个股技术类：**必须先有 `prompt_id`**，不要裸调 `get_mcp_analysis`
- 股价类问题：**禁止**默认 `capital_flow` 模板或 `get_capital_flow` 代替技术面
- `get_single_prompt_template` 返回精简列表（`brief`=intro.cn，无 template 正文）
- 结论必须基于 Tool 返回；失败如实说明，不编造行情

## 反模式

- 跳过 `search_code` 直接猜代码
- `get_single_prompt_template(type=tech)` **不传 tag** 拉全表
- 用 `web_search` 代替 `get_mcp_analysis` 做技术面

## 输出

结论先行；可注明周期与模板中文名；**不要**向用户展示 `prompt_id` / Mongo `_id`。附 1～3 条可操作建议（非必须调 Tool）。

**排版（与 SOUL 一致）**：`analysis_result` 仅作事实来源；给用户时须改写为 `##`/`###` + 列表，**勿照抄**其中的 `|表格|`、`---` 或粘连一行。标题与元信息（数据截至/现价）分行；列表每项单独一行；加粗成对使用，勿留下孤立 `**`。
