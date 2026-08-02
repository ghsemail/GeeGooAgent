---
name: stock-analysis
description: 个股技术面分析、指数分析、prompt_id、get_mcp_analysis、周线日线、模板列表、资金面、个股新闻。用户问「分析一下」「技术面」「用什么模板」「EMA/MACD 分析」时触发。
---

# 个股 / 指数分析（运行时）

## 适用 Toolset

`market` + `analyst_runtime`（默认 chat 已包含）

## 标准流程

1. **确认标的** — `search_code`（名称/代码模糊搜；特殊标的如 SpaceX 也在库内）
2. **选分析周期** — 未说明时 `clarify`：`daily` / `weekly` / `monthly` 等（与下文 `period` 枚举一致）
3. **选 Prompt 模板**（`type=tech` 时按用户意图选 **price / kline / flag**，勿默认 capital_flow）
   - 列表：`get_single_prompt_template`，`type` 取 `tech`（个股技术）/ `index`（指标）/ `fundamental`（基本面）
   - 用户问价格/走势/这周涨跌 → 在 tech 列表中优先 **flag > kline > price**；**直接采用**返回的 `recommended_for_price_trend.prompt_id`；**只有**用户点名资金时才选 capital_flow 模板
   - 已知指标名：`get_single_prompt_template_by_index`（`index`=variable 如 `EMA`，`period` 必填）
   - 从返回项取 **`prompt_id`**；`period` 与第 2 步一致
4. **执行分析** — `get_mcp_analysis`（`name`、`code`、`prompt_id`、`period`；`language` 默认 `cn`）
5. **可选增强**（用户关心资金/新闻时）
   - `get_capital_flow` / `get_capital_distribution`
   - `fetch_stock_news`

## 硬规则

- 个股技术类：**必须先有 `prompt_id`**，不要裸调 `get_mcp_analysis`
- `get_single_prompt_template` 只返回 **已启用**（`switch=true`）模板；运营新建模板后需 `switch_prompt_status`
- **竞品 / ETF 用户模板**走 `create_*_prompt_template`（analyze 服务），**不会**出现在 `get_single_prompt_template` 列表
- 结论必须基于 Tool 返回；失败如实说明，不编造行情

## 反模式

- 跳过 `search_code` 直接猜代码
- `period` 与 `prompt_id` 来源周期不一致
- 用 `web_search` 代替 `get_mcp_analysis` 做技术面

## 输出

结论先行；注明周期、模板名、`prompt_id`；附 1～3 条可操作建议（非必须调 Tool）。

**排版（与 SOUL 一致）**：`analysis_result` 仅作事实来源；给用户时须改写为 `##`/`###` + 列表，**勿照抄**其中的 `|表格|`、`---` 或粘连一行。
