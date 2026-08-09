---
name: intraday_stock
description: 盘中交易决策（Bot/Reminder 信号触发）
version: "1.0.0"
---

# intraday_stock

**信号触发**（非 cron）：TradingBot DCA/GRID（含 Reminder）在 `attitude.controll_switch=true` 时调用 `POST /v1/skills/run`（别名 `intraday`）。

**执行**：`internal/workflow/intraday.go`、`intraday_hourly_synth.go`、`memory/intraday_decision.go`；报告正文在 Go 中组装（无 `template.md`）。

## 必填入参

`code`、`stock_name`、`bot_id`、`bot_name`、`bot_type`、`frequency`、`trade_type`

## 步骤（按 Bot 动态裁剪）

1. `get_position` — 仅交易 Bot；Reminder 跳过且不展示持仓
2. `get_stock_daily_reports` — 仅 `attitude.switch=true` 时读盘前
3. `get_capital_distribution`
4. 小时级分析 — 按 `frequency`：`≤3m` 跳过；`3–10m` 仅价格 MCP；`≥10m` 价格+信号+K 线 bundle
5. 小时级 MCP 原文经 **LLM 摘要** 写入报告（非 OHLC 明细）
6. `get_current_price`
7. 规则引擎 + LLM 终审 → `save_local_report` + `create_stock_intraday_report`

硬规则示例：无持仓不卖；盘前高置信看空拦截买入；小时级与信号方向矛盾 → hold。

## 输出

- API：`createStockIntradayReport`
- 本地：`reports/<YYYYMMDD>/<code>-intraday.md`

## CLI

```bash
geegoo run intraday_stock --code 00700.HK --stock-name 腾讯控股 \
  --bot-id <id> --bot-name my-grid --bot-type GRID \
  --frequency 15m --trade-type 信号买入
```

Skills 详情中的 **Phase B** 为默认步骤快照；实际以运行时 `frequency` / `bot_type` / `attitude.switch` 为准。
