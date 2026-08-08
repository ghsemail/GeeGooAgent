---
name: premarket_stock
description: Per-stock pre-market reports for attitude-subscribed bots.
version: "1.0.0"
---

# premarket_stock

在对应市场报告生成后，为 `attitude.switch=true` 的 Bot/Reminder 标的生成个股盘前报告。每条报告仍绑定 `bot_id`。

## Run

```bash
geegoo run premarket_stock --market CN --config config.json
```

会先 `listReportUsers` 遍历该市场所有订阅用户，再逐用户执行。
