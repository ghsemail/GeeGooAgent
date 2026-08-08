# {{market_label}} 市场盘前报告

**生成时间**: {{timestamp}}
**市场**: {{market}}
**report_date**: {{report_date}}

---

## 一、指数走势（hourly MCP）

> 仅填写**当前市场**指数。无数据时写「暂无」，勿编造。

<!-- market=CN -->
### 上证指数 (000001.SH)
| 指标 | 值 |
|------|-----|
| 分析结论 | {{sh_analysis}} |

### 深证成指 (399001.SZ)
| 指标 | 值 |
|------|-----|
| 分析结论 | {{sz_analysis}} |
<!-- /market=CN -->

<!-- market=HK -->
### 恒生指数 (800000.HK)
| 指标 | 值 |
|------|-----|
| 分析结论 | {{hsi_analysis}} |
<!-- /market=HK -->

<!-- market=US -->
### 道琼斯工业指数 (^DJI.US)
| 指标 | 值 |
|------|-----|
| 分析结论 | {{dji_analysis}} |

### 纳斯达克综合指数 (^IXIC.US)
| 指标 | 值 |
|------|-----|
| 分析结论 | {{nasdaq_analysis}} |
<!-- /market=US -->

---

## 二、市场新闻摘要

> 仅 {{market}} 市场新闻，提取 3–5 条要点；**禁止输出原始 JSON**。

<!-- market=CN -->{{cn_market_news}}<!-- /market=CN -->
<!-- market=HK -->{{hk_market_news}}<!-- /market=HK -->
<!-- market=US -->{{us_market_news}}<!-- /market=US -->

---

## 三、市场综合预判

> 由 LLM 根据指数与新闻证据合成 `result` / `confidence` / `summary`，并写入完整 `report` 正文。

| 字段 | 值 |
|------|-----|
| 情绪 (result) | {{result}} |
| 置信度 (confidence) | {{confidence}} |
| 摘要 (summary) | {{summary}} |

### 主要风险
{{key_risks}}

### 今日关注点
{{key_watch_points}}

---

**报告生成**: geegoo-agent · skill `pre_market`
**下次更新**: 同市场下一交易日盘前 job
