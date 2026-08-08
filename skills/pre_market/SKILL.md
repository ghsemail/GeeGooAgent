---
name: pre_market
description: Global market pre-market report for CN, HK, or US.
version: "1.0.0"
---

# pre_market

按市场（A股 / 港股 / 美股）生成**全局一份**市场盘前报告：指数 hourly 分析 + 市场新闻，**LLM 合成**完整报告与 `result`/`confidence`/`summary`，入库 `market_pre_market_report`。

## Run

```bash
geegoo run pre_market --market CN --config config.json
```

Scheduler job 需带 `market` 参数（CN/HK/US）。
