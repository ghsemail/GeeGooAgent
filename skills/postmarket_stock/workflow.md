# Post-Market Analysis Workflow

**Trigger:** 17:00 Asia/Shanghai on trading weekdays for CN/HK (`0 17 * * 1-5`); US after close at 05:00 Tue–Sat (`0 5 * * 2-6`, Asia/Shanghai).

**Goal:** Summarize session price action, bot execution, and alignment with pre-market view.

**Output:** Local `{code}-postmarket.md` + `createStockPostmarketReport`.

## Phase A

1. `check_trading_day`
2. `get_report_bot_codes`

## Phase B (per bot stock)

1. `list_today_stock_postmarket_reports` — skip if already reported
2. One parallel `get_hourly_analysis_bundle` call (price / signal / kline hourly MCP)
3. `get_bot_log_by_type` (`DCA` or `GRID`)
4. `get_stock_daily_reports` — `stock_premarket[0]` for `vs_stock_premarket`
5. `get_current_price` — `change_pct` fallback
6. Compute `session_bias` from `change_pct` (not LLM)
7. `save_local_report` + `create_stock_postmarket_report`

## Field Rules

| Field | Rule |
| --- | --- |
| `session_bias` | `change_pct` >1% → bullish; <-1% → bearish; else neutral |
| `vs_stock_premarket` | Compare pre `result` with `session_bias` |
| `stock_premarket_report_id` | From API `stock_premarket[0].report_id` when present |

## Run

```bash
geegoo run postmarket_stock --config config.json
geegoo run postmarket_stock --dry-run
```
