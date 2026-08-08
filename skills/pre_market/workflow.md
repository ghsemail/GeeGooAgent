# Market Pre-Market Workflow (`pre_market`)

**Scope:** One run handles **exactly one market** — `CN`, `HK`, or `US`. The market is set by scheduler job `market` or CLI `--market`.

**Goal:** Produce **one global market pre-market report per market per trading day**.

**Output:**

| Sink | Path / API |
|------|------------|
| API | `create_market_pre_market_report` → `market_pre_market_report` collection |
| Local MD | `{workspace_root}/reports/<YYYYMMDD>/market-<MARKET>-market-premarket.md` |

**Out of scope** (see `pre_market_stock`):

- `get_report_bot_codes`
- Per-stock `create_pre_market_report`
- `bot_id` binding

Tool allowlists and step IDs: `manifest.yaml`. API routing: `rules/api-routing.md`.

---

## Scheduler (Asia/Shanghai, weekdays)

| Job | Skill | Market | Cron | Notes |
|-----|-------|--------|------|-------|
| `pre_market_cn` | `pre_market` | CN | `0 8 * * 1-5` | A-share market report |
| `pre_market_stock_cn` | `pre_market_stock` | CN | `10 8 * * 1-5` | Stocks, 10 min later |
| `pre_market_hk` | `pre_market` | HK | `0 9 * * 1-5` | HK market report |
| `pre_market_stock_hk` | `pre_market_stock` | HK | `10 9 * * 1-5` | Stocks |
| `pre_market_us` | `pre_market` | US | `0 21 * * 1-5` | US market report |
| `pre_market_stock_us` | `pre_market_stock` | US | `10 21 * * 1-5` | Stocks |

`pre_market_stock` **must** run after the matching `pre_market` job for the same market.

---

## Run

```bash
geegoo run pre_market --market CN --config config.json
geegoo run pre_market --market HK --config config.json
geegoo run pre_market --market US --config config.json
```

---

## Phase A (only phase)

### 1. Trading day check

Tool: `check_trading_day`

Reference codes by market:

| Market | Code |
|--------|------|
| CN | `000001.SZ` |
| HK | `00700.HK` |
| US | `AAPL.US` |

- `is_trading_day: false` → write execution log and **stop** (no report).
- `is_trading_day: true` → continue.

### 2. Index analysis (hourly MCP)

Tool: `get_mcp_analysis` with `period=hourly`, `language=cn`.

Indices collected **for the current market only**:

| Market | Indices |
|--------|---------|
| CN | 上证指数 `000001.SH`, 深证成指 `399001.SZ` |
| HK | 恒生指数 `800000.HK` |
| US | 道琼斯 `^DJI.US`, 纳斯达克 `^IXIC.US` |

Do **not** fetch other markets' indices in the same run.

### 3. Market news

Tool: `fetch_market_news`

```json
{"market": "<CN|HK|US>", "limit": 8}
```

Only news for the **current** market. Summarize 3–5 headlines; do not dump raw JSON into the report.

### 4. Persist

1. `save_local_report` — `report_type=market-premarket`, code `market-<MARKET>`
2. `create_market_pre_market_report` — fields: `market`, `report`, optional `summary` / `result` / `confidence`

Report body follows `template.md` (single-market sections only).

### 5. LLM synthesis (required when gateway configured)

Before persist, build a rule-based **draft** from index + news evidence, then call the report synthesizer:

1. Input: `market`, draft markdown, `template.md`, evidence refs
2. Output JSON: `report`, `result`, `confidence`, `summary`
3. LLM must only use captured evidence — no invented prices or headlines
4. On synthesis failure → fall back to draft + rule-based `result`/`confidence`

---

## Relationship to `pre_market_stock`

```text
pre_market (market=CN)     →  market_pre_market_report (CN, today)
        ↓ 10 min
pre_market_stock (market=CN)  →  get_market_pre_market_report
                             →  get_report_bot_codes (filtered by market)
                             →  per-stock create_pre_market_report
```

Stock reports embed the market report as **Market Context**; they do not re-fetch all three markets.

---

## Execution log

Tool: `write_execution_log`

Path: `{workspace_root}/reports/<YYYYMMDD>/execution-log.md`

Example:

```text
[08:00:01] check_trading_day(CN) -> success(is_trading_day=true)
[08:00:45] index_000001.SH -> success
[08:01:10] market_news_cn -> success
[08:01:20] create_market_pre_market_report(CN) -> success
[08:01:21] phase_a_complete -> ok
```
