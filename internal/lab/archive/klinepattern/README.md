# K-line pattern display (archived 2026-10-20)

Diagnose **display-only** K-line pattern annotations: whitelisted TA-Lib pattern hits on trigger bars, optional prev2/prev/anchor three-bar windows, and MTF weighted bias JSON (PRs [#54](https://github.com/ghsemail/GeeGooAgent/pull/54), [#58](https://github.com/ghsemail/GeeGooAgent/pull/58), [#59](https://github.com/ghsemail/GeeGooAgent/pull/59)).

## Why parked

Production signal diagnose should not pay MTF probe cost or show pattern fields by default. Offline evaluation showed MTF weighted bias was not predictive; the 240m ladder slot often returned empty series. Further 240m work is paused.

## Production default

- Config JSON: `kline_pattern_display` — **false** when unset
- Env: `GEEGOO_KLINE_PATTERN_DISPLAY=true` to override
- Runtime switch: `strategydev.ConfigureKlinePatternDisplayEnabled` (wired from `AppConfig` in `internal/app/app.go`)

When off, `runDiagnose` / signal-diagnose evaluate paths skip MTF probes and do not attach `kline_pattern_*` fields to episodes or triggers.

## Lab / re-enable

1. Set `GEEGOO_KLINE_PATTERN_DISPLAY=true` or `"kline_pattern_display": true` in agent config.
2. Run archived validator (needs Signal + MCP tokens):

   ```bash
   go run ./internal/lab/archive/klinepattern/cmd/validate
   ```

3. Import this package from lab experiments; production hooks live in `internal/workflow/strategydev/kline_pattern_bridge.go`.

Git tag: `archive/kline-pattern-2026-10-20`.
