# Hermes to GeeGoo Agent Cutover Checklist

This checklist records the manual cutover steps from Hermes scheduling to the Go-native GeeGoo Agent. Each item now includes the exact command to verify it.

See `deploy/hermes-parity-roadmap.md` for the P1–P8 optimization roadmap that delivered the capabilities below.

## Preconditions

- [ ] `go test ./...` passes.
  Verify: `cd D:\Geegoo\GeeGooAgent && go test ./...` (expect all `ok`)
- [ ] `go build ./cmd/geegoo` passes.
  Verify: `go build -o /tmp/geegoo ./cmd/geegoo`
- [ ] Server smoke checks in `tests/smoke/README.md` pass.
  Verify: `geegoo doctor` (all `[OK]`)
- [ ] `/etc/geegoo-agent/config.json` has mode `600`.
  Verify: `stat -c '%a' /etc/geegoo-agent/config.json` → `600`
- [ ] Secrets are not committed to git.
  Verify: `git log --all -p | grep -E "mcp_HVT|sk-[A-Za-z0-9]{20}"` returns nothing

## Parallel Verification

- [ ] `geegoo-agent-pre-market.timer` is enabled (or `geegoo scheduler run` is running).
  Verify: `systemctl is-enabled geegoo-agent-pre-market.timer` OR `pgrep -af 'geegoo scheduler run'`
- [ ] GeeGoo Agent and the old scheduler run in parallel for at least one trading day.
  Verify: both produced reports for the same date — `geegoo verify --date <D> --codes 00700.HK,000001.SZ,SPACEX.US` returns PASS and the old Hermes run also completed.
- [ ] Per-stock local `{code}-premarket.md` files exist.
  Verify: `ls ~/.geegoo/data/reports/<D>/*-premarket.md` lists one per stock
- [ ] GeeGoo API has `createPreMarketReport` records.
  Verify: `geegoo verify --date <D> --codes 00700.HK` prints non-empty report cards
- [ ] `bot_id`, `bot_name`, and `bot_type` are non-empty.
  Verify: `geegoo verify` completeness matrix shows `bot_id` / `bot_name` / `bot_type` = 100%
- [ ] `execution-log.md` records the complete workflow.
  Verify: `grep -c '^- \[' ~/.geegoo/data/<D>/execution-log.md` ≥ expected step count
- [ ] Sample reports are structurally comparable with the old output.
  Verify: diff a GeeGoo `-premarket.md` against the Hermes-era equivalent for the same stock/date; result/suggestion enums must match, reason must be ≥80 chars with evidence refs.
- [ ] Evidence refs are present and traceable.
  Verify: `geegoo verify` shows `evidence_refs` 100%; spot-check one ref id via `sqlite3 ~/.geegoo/data/geegoo.db "SELECT id,source,summary FROM evidence_records WHERE id='<ev_id>'"`

## Cutover

- [ ] Confirm `check_trading_day` returns a valid trading-day decision.
  Verify: `geegoo doctor` shows `[OK] GeeGooBot mcp checkTradingDay`
- [ ] Disable the old Hermes pre-market cron manually and record the original cron line for rollback.
  Verify: `crontab -l | grep -i pre_market` (capture output to rollback notes)
- [ ] Keep only `geegoo-agent-pre-market.timer` (or `geegoo scheduler run`) active.
  Verify: only one of the two is running — `systemctl is-active geegoo-agent-pre-market.timer` XOR `pgrep -af 'geegoo scheduler run'`
- [ ] Observe the first independent run with `journalctl`, local reports, and API records.
  Verify: `journalctl -u geegoo-agent-pre-market -S today | tail -50` + `geegoo verify --date <D> --codes <sample>` PASS

## Rollback

- [ ] Re-enable the old Hermes pre-market cron.
  Verify: `crontab -l | grep -i pre_market` shows the restored line
- [ ] Disable the GeeGoo timer / scheduler.
  Verify: `systemctl disable --now geegoo-agent-pre-market.timer` AND `pkill -f 'geegoo scheduler run'`
- [ ] Preserve `/var/lib/geegoo-agent` (or `~/.geegoo/data`) for troubleshooting.
  Verify: `ls -la ~/.geegoo/data/geegoo.db ~/.geegoo/data/reports/` still present

## Acceptance Thresholds (quantified)

| Metric | Target |
|---|---|
| `go test ./...` | 100% packages `ok` |
| `geegoo doctor` | all `[OK]`, zero `[FAIL]` |
| `bot_id`/`bot_name`/`bot_type` non-empty rate | 100% |
| `result` enum validity | 100% in {long, short, neutral} |
| `confidence` enum validity | 100% in {high, medium, low, review_required} |
| `suggestion` enum validity | 100% in {buy, sell, hold, watch_long, reduce_or_avoid} |
| `reason` length | ≥80 chars |
| `evidence_refs` present rate | 100% |
| Supervisor verdict on a clean trading day | `pass` |
| Resume idempotency | re-running `geegoo resume --session <id>` skips completed step keys |
