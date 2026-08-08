# Stock Pre-Market Workflow (`pre_market_stock`)

**Scope:** One run handles **one market** (`CN` / `HK` / `US`). Generates **per-stock** pre-market reports for bots with `attitude.switch=true`.

**Prerequisite:** Matching `pre_market` job for the same market must have completed (market report available via `get_market_pre_market_report`).

**Goal:** Per-stock pre-market prediction with `bot_id` binding.

**Output:**

| Sink | Path / API |
|------|------------|
| API | `create_pre_market_report` (includes `market_pre_market_report_id`) |
| Local MD | `{workspace_root}/reports/<YYYYMMDD>/<code>-premarket.md` |

Tool allowlists: `manifest.yaml`. Report format: `rules/report-format.md` (stock section).

---

## Scheduler

Runs **10 minutes after** the market `pre_market` job for the same market (see `pre_market/workflow.md`).

```bash
geegoo run pre_market_stock --market CN --config config.json
```

---

## Phase A: Market context + stock list

### 1. Load market report

Tool: `get_market_pre_market_report`

```json
{"market": "<CN|HK|US>", "report_date": "YYYY-MM-DD"}
```

Inject body into each stock report as **Market Context**. If 404, log warning and continue with empty context (degraded).

### 2. Stock discovery

Tool: `get_report_bot_codes`

Returns bots subscribed for reports; workflow filters to stocks in the **current market** only.

Required fields per row: `code`, `stock_name`, `bot_id`, `bot_name`, `bot_type`.

Deduplicate by `code`. Empty list → write execution log and stop Phase B.

---

## Phase B: Per stock

For each stock in the current market:

1. `list_today_reports` — skip if already reported today
2. `fetch_stock_news`
3. `get_capital_flow` (`period=DAY`)
4. `get_capital_distribution`
5. `get_mcp_analysis` (`period=weekly`)
6. `get_bot_yesterday_attitude`
7. Synthesize with `template.md` (stock sections; market overview comes from Phase A)
8. `save_local_report` → `{code}-premarket.md`
9. `create_pre_market_report` with `market_pre_market_report_id`

Unsupported APIs: note explicitly in report; mark step skipped in execution log.

---

## Execution log

```text
[08:10:01] get_market_pre_market_report(CN) -> success
[08:10:03] get_report_bot_codes -> success(8 stocks)
[08:11:20] create_pre_market_report(600519.SH) -> success
[08:14:10] workflow -> complete
```
