---
name: premarket_market
description: 市场盘前报告（CN/HK/US 各一份/日）
version: "1.0.0"
---

# premarket_market

按市场（A股 / 港股 / 美股）生成**全局一份**市场盘前报告：指数 hourly 分析 + 市场新闻 → LLM 合成正文与 `result`/`confidence`/`summary` → 入库 `create_market_premarket_report`。

**执行**：由 `internal/workflow/market.go` 驱动；**报告正文模板**见同目录 `template.md`（运行时加载）。

## 触发

| Job | Market | Cron（工作日） |
|-----|--------|----------------|
| `premarket_market_cn` | CN | `0 8 * * 1-5` |
| `premarket_market_hk` | HK | `0 9 * * 1-5` |
| `premarket_market_us` | US | `0 21 * * 1-5` |

`premarket_stock` 应在同市场盘前报告 **10 分钟后**运行。

## 输出

- API：`create_market_premarket_report`
- 本地：`reports/<YYYYMMDD>/market-<MARKET>-market_premarket.md`

## CLI

```bash
geegoo run premarket_market --market CN --config config.json
```

Skills 模块中的 **Phase A/B 步骤**与 **调度计划** 来自 Go 注册表（与线上执行一致）。
