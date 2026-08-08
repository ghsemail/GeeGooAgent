<!-- 元数据仅作维护说明，勿写入 report 正文：code={{code}} market={{market}} bot={{bot_id}} -->

## 市场背景

> 引用当日 `premarket_market` 摘要（`get_market_premarket_report`）。勿重复三市场指数。

{{market_report_summary}}

---

## 个股新闻

> 3–5 条要点；**禁止罗列发布时间或 🕐 时间戳**。

{{stock_news}}

---

## 资金流向与分布

### 主力资金（日）

{{capital_flow}}

### 资金分布

{{capital_distribution}}

禁止输出 raw JSON；需有定量结论（如「主力净流入 X 亿」）。

---

## 周线技术分析

> `getMCPAnalysis` `period=weekly` **不返回 RSI/MACD**。无数据填「暂无」，勿编造。用自然段描述支撑/阻力/趋势，**禁止 Markdown 表格**。

{{weekly_analysis}}

---

## Bot 盘前态度

上一交易日态度（服务端回溯最近 7 天）：

{{bot_attitude}}

---

## 综合研判

> 多维度综合（市场背景 × 新闻 × 资金 × 周线 × Bot 态度）。用自然段描述，**禁止在正文重复 result/confidence/suggestion**（由 API 字段与 App 概要区展示）。

{{reason}}

### 今日重点关注

{{key_watch_points}}

### 风险提示

{{risk_warnings}}

---

*报告由 GeeGoo 智能体个股盘前 skill 生成*
