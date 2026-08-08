# {{market_label}} 市场盘前报告

**生成时间**: {{timestamp}}
**市场**: {{market}}
**report_date**: {{report_date}}

---

## 指数概览

> 仅填写**当前市场**指数。无数据时写「暂无」，勿编造。每个指数用 1–2 句自然段描述，**禁止 Markdown 表格**。

<!-- market=CN -->
### 上证指数 (000001.SH)
{{sh_analysis}}

### 深证成指 (399001.SZ)
{{sz_analysis}}
<!-- /market=CN -->

<!-- market=HK -->
### 恒生指数 (800000.HK)
{{hsi_analysis}}
<!-- /market=HK -->

<!-- market=US -->
### 道琼斯工业指数 (^DJI.US)
{{dji_analysis}}

### 纳斯达克综合指数 (^IXIC.US)
{{nasdaq_analysis}}
<!-- /market=US -->

---

## 市场新闻解读

> 仅 {{market}} 市场新闻。提取 3–5 条要点（**禁止罗列发布时间、🕐 时间戳**）；必须给出 **新闻面判断**（偏多/偏空/中性 + 一句话理由）。

**新闻面判断**：{{news_bias}}

<!-- market=CN -->{{cn_market_news}}<!-- /market=CN -->
<!-- market=HK -->{{hk_market_news}}<!-- /market=HK -->
<!-- market=US -->{{us_market_news}}<!-- /market=US -->

---

## 市场综合判断

> 由 LLM 根据指数与新闻证据合成。用自然段描述综合结论，**禁止 Markdown 表格**；**不要在正文中重复市场情绪或置信度**（已由 App 概要区展示）。

{{summary}}

### 主要风险
{{key_risks}}

### 今日关注
{{key_watch_points}}

---

*报告由 GeeGoo 智能体市场盘前 skill 生成*
