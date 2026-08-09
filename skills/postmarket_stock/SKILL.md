---
name: postmarket_stock
description: 个股盘后总结（交易日 scheduler）
version: "1.0.0"
---

# postmarket_stock

交易日 scheduler 触发，汇总当日盘面、Bot 成交与盘前对照。

**执行**：`internal/workflow/postmarket.go`；`session_bias` / `vs_stock_premarket` 在 Go 中计算（非 LLM）。

## 触发

| Job | Cron（Asia/Shanghai） |
|-----|------------------------|
| `postmarket_stock_cn` / `_hk` | `0 17 * * 1-5` |
| `postmarket_stock_us` | `0 5 * * 2-6` |

## Phase B（逐股）

1. `list_today_stock_postmarket_reports` — 已报则跳过
2. `get_hourly_analysis_bundle`
3. `get_bot_log_by_type`（DCA / GRID）
4. `get_stock_daily_reports` — 盘前对照
5. `get_current_price` — `change_pct`
6. `save_local_report` + `create_stock_postmarket_report`

## 字段规则

| 字段 | 规则 |
|------|------|
| `session_bias` | 涨跌幅 >1% 偏多；<-1% 偏空；否则中性 |
| `vs_stock_premarket` | 盘前 `result` 与 `session_bias` 对照 |

## CLI

```bash
geegoo run postmarket_stock --config config.json
```

步骤与调度见 Skills 详情（Go 注册表）。
