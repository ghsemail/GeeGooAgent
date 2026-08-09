---
name: premarket_stock
description: 个股盘前报告（attitude 订阅标的）
version: "1.0.0"
---

# premarket_stock

在对应市场 `premarket_market` 完成后，为 `attitude.switch=true` 的 Bot/Reminder 标的生成个股盘前报告；每条绑定 `bot_id`。

**执行**：`internal/workflow/stock.go` 等；**LLM 正文模板**见同目录 `template.md`（运行时加载）。

## 触发

| Job | Market | Cron |
|-----|--------|------|
| `premarket_stock_cn` | CN | `10 8 * * 1-5` |
| `premarket_stock_hk` | HK | `10 9 * * 1-5` |
| `premarket_stock_us` | US | `10 21 * * 1-5` |

## 输出

- API：`create_stock_premarket_report`（含 `market_premarket_report_id`）
- 本地：`reports/<YYYYMMDD>/<code>-premarket.md`

## CLI

```bash
geegoo run premarket_stock --market CN --config config.json
```

按市场 `listReportUsers` 逐用户执行。步骤与调度见 Skills 详情中的 **Phase / 调度计划**（Go 代码为准）。
