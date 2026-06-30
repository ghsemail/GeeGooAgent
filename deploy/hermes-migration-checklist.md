# Hermes to GeeGoo Agent Cutover Checklist

This checklist records the manual cutover steps from Hermes scheduling to the Go-native GeeGoo Agent systemd timer.

## Preconditions

- [ ] `go test ./...` passes.
- [ ] `go build ./cmd/geegoo` passes.
- [ ] Server smoke checks in `tests/smoke/README.md` pass.
- [ ] `/etc/geegoo-agent/config.json` has mode `600`.
- [ ] Secrets are not committed to git.

## Parallel Verification

- [ ] `geegoo-agent-pre-market.timer` is enabled: `systemctl is-enabled geegoo-agent-pre-market.timer`.
- [ ] GeeGoo Agent and the old scheduler run in parallel for at least one trading day.
- [ ] Per-stock local `{code}-premarket.md` files exist.
- [ ] GeeGoo API has `createPreMarketReport` records.
- [ ] `bot_id`, `bot_name`, and `bot_type` are non-empty.
- [ ] `execution-log.md` records the complete workflow.
- [ ] Sample reports are structurally comparable with the old output.

## Cutover

- [ ] Confirm `check_trading_day` returns a valid trading-day decision.
- [ ] Disable the old Hermes pre-market cron manually and record the original cron line for rollback.
- [ ] Keep only `geegoo-agent-pre-market.timer` active.
- [ ] Observe the first independent run with `journalctl`, local reports, and API records.

## Rollback

- [ ] Re-enable the old Hermes pre-market cron.
- [ ] Disable the GeeGoo timer: `systemctl disable --now geegoo-agent-pre-market.timer`.
- [ ] Preserve `/var/lib/geegoo-agent` for troubleshooting.
