# Intraday Trade Decision Workflow

**Trigger:** signal from Bot/Reminder (not cron).

**Goal:** Decide whether to approve the current buy/sell signal and persist an intraday decision report.

**Output:** Local `{code}-intraday.md` + `createIntradayTradeDecisionReport`.

Tool allowlist and step IDs are in `manifest.yaml`. Decision rules follow geegoo `intraday-workflow.md` Step 5.5.

## Required Inputs

| Field | Description |
| --- | --- |
| `code` | e.g. `00700.HK` |
| `stock_name` | Display name |
| `bot_id` | Triggering bot/reminder id |
| `bot_name` | Bot name (`botname`) |
| `bot_type` | `DCA` / `GRID` / `*Reminder` |
| `frequency` | e.g. `5m`, `15m`, `1h` |
| `trade_type` | e.g. `信号买入` / `信号卖出` |

## CLI

```bash
geegoo run intraday --code 00700.HK --stock-name 腾讯控股 \
  --bot-id <id> --bot-name my-grid --bot-type GRID \
  --frequency 5m --trade-type 信号买入
```

Environment variables `GEEGOO_INTRADAY_*` are also supported.

## Steps (per stock)

1. `get_position` — skip constraint for `*Reminder`
2. `get_stock_daily_reports` — read `pre_market[0]`
3. `get_capital_distribution` — skip A-shares (`.SH`/`.SZ`)
4. `get_mcp_analysis` hourly — frequency rules in manifest
5. `get_current_price` → `get_ticker` fallback
6. Rule-based `result` / `confidence` / `reason` (≥80 chars)
7. `save_local_report` + `create_intraday_report`

`update_intraday_report` (Step 7 in geegoo) is executed by the trade executor after fills, not by this workflow.

## Execution Log

`{workspace}/reports/<YYYYMMDD>/execution-log.md`
