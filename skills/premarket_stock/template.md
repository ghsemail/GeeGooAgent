<!-- 元数据仅作维护说明，勿写入 report 正文：code={{code}} market={{market}} bot={{bot_id}} -->

## 市场背景

> 引用当日 `premarket_market` 摘要（`get_market_premarket_report`）。勿重复三市场指数。

{{market_report_summary}}

---

## 个股新闻

> 先写 **新闻综述**（1–2 句），再列 3–5 条标题要点；**禁止**股票代码、链接、🕐 时间戳。

{{stock_news}}

---

## 资金流向与分布

> 用自然段描述主力净流入/流出与超大/大/中/小单结构；金额用「亿/万」书写，**禁止** `main_in_flow=` 等参数名。

{{capital_flow}}

{{capital_distribution}}

---

## 周线技术分析

> 与上文同样用自然段 + 短列表；**禁止** `#` 标题、Markdown 表格、照抄 MCP 原文排版。

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
