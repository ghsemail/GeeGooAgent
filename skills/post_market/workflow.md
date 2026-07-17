# Post-Market Analysis Workflow

**Trigger:** 17:00 Asia/Shanghai on trading weekdays (A-share/HK); US market uses separate cron per geegoo skill.

**Goal:** Summarize session price action, bot execution, and alignment with pre-market view.

**Output:** Local `{code}-postmarket.md` + `createPostMarketReport`.

## Phase A

1. `check_trading_day`
2. `get_report_bot_codes`

## Phase B (per bot stock)

1. `list_today_post_market_reports` — skip if already reported
2. Three `get_mcp_analysis` hourly calls (price / signal / kline)
3. `get_bot_log_by_type` (`DCA` or `GRID`)
4. `get_stock_daily_reports` — `pre_market[0]` for `vs_pre_market`
5. `get_current_price` — `change_pct` fallback
6. Compute `session_bias` from `change_pct` (not LLM)
7. `save_local_report` + `create_post_market_report`

## Field Rules

| Field | Rule |
| --- | --- |
| `session_bias` | `change_pct` >1% → bullish; <-1% → bearish; else neutral |
| `vs_pre_market` | Compare pre `result` with `session_bias` |
| `pre_market_report_id` | From API `pre_market[0].report_id` when present |

## Run

```bash
geegoo run post_market --config config.json
geegoo run post_market --dry-run
```
