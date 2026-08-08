---
name: premarket_market
description: Global market pre-market report for CN, HK, or US.
version: "1.0.0"
---

# premarket_market

按市场（A股 / 港股 / 美股）生成**全局一份**市场盘前报告：指数 hourly 分析 + 市场新闻，**LLM 合成**正文（指数概览 / 新闻解读 / 综合判断）与 API 字段 `result`/`confidence`/`summary`，入库 `market_premarket_report`。

正文规范见 `template.md` 与 `rules/report-format.md`（无表格、无时间戳、正文不写情绪/置信度）。

## Run

```bash
geegoo run premarket_market --market CN --config config.json
```

Scheduler job 需带 `market` 参数（CN/HK/US）。
