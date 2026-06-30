# Pre-Market Workflow

**Trigger time:** 08:00 Asia/Shanghai on trading weekdays by the default systemd timer.

**Goal:** Generate pre-market prediction reports for configured bot stocks.

**Output:** One local Markdown report per stock plus persisted GeeGoo API records.

Tool allowlists and step IDs are defined in `manifest.yaml`. API routing is defined in `rules/api-routing.md`.

## Precondition: Trading Day Check

Before any data collection step, the workflow must run `check_trading_day`.

Tool: `check_trading_day`

```http
POST /checkTradingDay
Authorization: Bearer <sk-api-key>
Content-Type: application/json

{"mcp_token": "<mcp_token>", "code": "00700.HK"}
```

Decision rules:

- `is_trading_day: true` continues to stock discovery and report generation.
- `is_trading_day: false` ends the workflow and writes an execution log.

Recommended market reference codes: `00700.HK`, `AAPL.US`, `600519.SH`.

## Stock Discovery

Tool: `get_report_bot_codes`

```http
POST /getReportBotCodes
Content-Type: application/json

{"mcp_token": "<mcp_token>"}
```

Required fields:

| Field | Description |
| --- | --- |
| `code` | Stock code. |
| `stock_name` | Stock name. |
| `bot_id` | Bot document `_id`. |
| `bot_name` | Bot name. |
| `bot_type` | Bot type, such as `DCA`. |

Deduplicate by `code`. If the list is empty, skip per-stock work and write an execution log.

## Phase A: Shared Market Data

1. Run `get_mcp_analysis` for market indexes with `period=hourly`.
2. Fetch market news through the Go news/search tool implementation.
3. Summarize US, CN, and HK market context for the report.

## Phase B: Per-Stock Data

For each stock returned by `get_report_bot_codes`:

1. Fetch stock news through the Go news/search tool implementation.
2. Run `get_capital_flow` with `period=DAY`.
3. Run `get_capital_distribution`.
4. Run weekly `get_mcp_analysis` with `period=weekly`.
5. Run `get_bot_yesterday_attitude`.
6. Synthesize the report with `template.md`.
7. Persist with `create_pre_market_report`.
8. Save local Markdown to `{workspace_root}/reports/<YYYYMMDD>/<code>-premarket.md`.

For unsupported market-specific APIs, write an explicit unavailable note in the report and mark the step as skipped in the execution log.

## Execution Log

Tool: `write_execution_log`

Path:

```text
{workspace_root}/reports/<YYYYMMDD>/execution-log.md
```

Example:

```text
[08:00:01] check_trading_day -> success(is_trading_day=true)
[08:00:03] get_report_bot_codes -> success(12 stocks)
[08:01:20] create_pre_market_report(00700.HK) -> success
[08:04:10] workflow -> complete
```
