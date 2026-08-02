# MCP 分析路由：技术分析 / 指标分析 / 基本面分析

本文档是 GeeGoo Agent **选用分析模板与调用 get_mcp_analysis 的单一事实来源**。
由 `chatprompt.AnalysisRouting()` 嵌入系统 prompt；修改后需重新部署 agent-runtime。

---

## 1. 三类分析（产品语言 ↔ API）

| 产品叫法 | get_single_prompt_template 的 type | 回答什么问题 | 库内模板 type 字段（TemplateType） | 典型用户说法 |
|----------|-----------------------------------|--------------|-----------------------------------|--------------|
| **技术分析** | **tech** | 走势、趋势、K 线形态、价格结构、资金流向、竞品/ETF 对比 | price, kline, flag, capital_flow, industry, competitor, etf, template | 「技术面」「走势」「趋势」「形态」「涨跌怎么看」「这周价格」 |
| **指标分析** | **index** | 某一技术指标的多空结论（看多/看空/持有） | index（variable=MACD/EMA/KDJ/RSI 等） | 「MACD」「EMA 信号」「RSI 多空」「金叉死叉」「某指标信号」 |
| **基本面分析** | **fundamental** | 财报、估值、盈利、风险、行业基本面 | basic, risk, financial, industry | 「基本面」「财报」「估值」「PE/PB」「盈利能力」「行业地位」 |

**易混点（必须遵守）：**

- API 参数 `type=index` 表示 **指标分析**（MACD 类），**不是**「恒生指数 / 上证指数」等指数标的。
- 分析任意标的（含指数成分股）均先 `search_code`，再按用户意图选上表 type。
- 用户只说「分析趋势 / 技术面 / 走势」时 **禁止默认 MACD**；应走 **tech**，在模板列表中优先选名称/简介含「趋势」「K 线」「价格」的项（优先级：flag > kline > price）。

---

## 2. 路由决策（按用户意图）

```
用户要分析某只股票
    │
    ├─ 提到 MACD / EMA / KDJ / RSI / 「指标信号」「金叉死叉」
    │       → type=index
    │       → get_single_prompt_template(type=index, period=…)
    │          或 get_single_prompt_template_by_index(index=MACD, period=…)
    │
    ├─ 提到基本面 / 财报 / 估值 / 盈利 / 行业
    │       → type=fundamental
    │       → get_single_prompt_template(type=fundamental, period=…)
    │
    └─ 技术面 / 趋势 / 走势 / 形态 / 涨跌 / **价格** / **这周价格**（未点名具体指标）【默认】
            → type=tech
            → get_single_prompt_template(type=tech, period=…)
            → 在返回列表中选最匹配模板（**flag 趋势 > kline 形态 > price 价格**）
            → **禁止**仅因列表里有「资金流向」就选 tag=capital_flow 模板；资金是补充项，不是价格/走势主分析
            → **优先**使用 `get_single_prompt_template` 返回的 **`recommended_for_price_trend.prompt_id`**（列表已重排，capital_flow 靠后）
```

**周期 period：**

| 用户说法 | 建议 period |
|----------|-------------|
| 默认 / 日线趋势 | daily |
| 这周 / 周线 | weekly |
| 短线 / 小时 | hourly |
| 分钟级 | minutes |

---

## 3. 标准工具链（必须按序）

1. **search_code** — 确认 code、name（如 00700.HK、01810.HK）
2. **get_single_prompt_template** 或 **get_single_prompt_template_by_index** — 取 `prompt_id`（列表仅含 `name_cn` / **`brief`（来自 intro）** / `tag`，**不含** `template` 正文）
3. **get_mcp_analysis**(name, code, prompt_id, period) — period 必填；经 GeeGooBot mcp-api → analyze-api，**勿直连 :3230**
4. 可选组合（有数据再用，禁止编造）：
   - **get_current_price** — 现价
   - **get_capital_flow** / **get_capital_distribution** — 资金（GeeGooData）
   - **fetch_stock_news** / **fetch_market_news** — 新闻

若 get_mcp_analysis 返回数据字段不全（仅 trade_date 等），如实说明限制，可改用 **get_current_price** / 新闻做补充，**不得**默认改用 capital_flow 模板或 `get_capital_flow` 冒充股价分析（除非用户明确问资金）。

---

## 4. 技术分析（type=tech）细目

| TemplateType | 用途 | 何时优先 |
|--------------|------|----------|
| **flag** | 综合趋势类信号 | 用户问「趋势」「走势方向」 |
| **kline** | K 线形态 | 用户问「形态」「K 线」 |
| **price** | 价格走势结构 | 用户问「价格」「涨跌」「这周价格」「分析价格」 |
| **capital_flow** | 资金流向（模板内嵌） | **仅**用户明确问「资金」「主力」「净流入」时；勿替代 price/kline/flag |
| competitor / etf / industry | 竞品、ETF、行业对比 | 用户明确对比需求 |

---

## 5. 指标分析（type=index）细目

- 每条模板对应 **一个** 技术指标（Mongo `variable` 字段，如 MACD、EMA）。
- 输出侧重：多空信号、金叉死叉、指标数值解读。
- **get_single_prompt_template_by_index**：用户已指明指标名时使用（如 index=MACD, period=daily）。
- 勿将「分析腾讯趋势」类泛化问题默认路由到 MACD。

---

## 6. 基本面分析（type=fundamental）细目

| TemplateType | 用途 |
|--------------|------|
| **basic** | 基本面概览 |
| **financial** | 财务数据解读 |
| **risk** | 风险因素 |
| **industry** | 行业地位与同业 |

---

## 7. 与 Bot 交易信号的区别

- 本文档路由的是 **LLM 分析模板**（single_prompt_template + get_mcp_analysis）。
- **Bot 买卖信号**（如 custom.index=MACDResonance）属于 **getBotSignal / 自定义策略**，不在此路由表内；用户要「建 Bot / 回测信号」时走策略与 loopback 工具链。

---

## 9. 用户可见回复格式（与 SOUL 一致）

上文路由表、工具链表格 **仅供你理解路由与参数**，不要在给用户的最终答复中照抄 `|...|` 宽表格或 `---` 分隔线。

- `get_mcp_analysis` 返回的 `analysis_result` 可能含表格或粘连排版；**引用时必须改写**为 SOUL 规定的格式：`##`/`###` 标题、` - ` 列表、`**字段**：值`。
- 多字段对比（如关键价位、资金维度）优先用「**标签**：说明」列表，不要用 Markdown 表格。

---

## 8. 维护说明

- 模板列表以 Mongo `single_prompt_template`（switch=true）为准；运营新增模板后需启用才会出现在 get_single_prompt_template。
- 若产品更名「指标分析 / 技术分析」显示文案，**API type 枚举不变**（仍为 tech / index / fundamental）。
