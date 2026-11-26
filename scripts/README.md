# GeeGooAgent ops scripts

This directory holds Agent-related deployment, probe, verification, and workflow scripts.

## Workflow Skills docs (`skills/`)

Pre-market / intraday / post-market keep only:

| File | Purpose |
|------|------|
| `SKILL.md` | Skills module presentation + Chat procedural memory |
| `template.md` | **Pre-market only** (market/stock); loaded at runtime by LLM synthesis |
| Go `internal/workflow/*.go` | Actual execution steps (Phase / scheduling in the Skills details) |

`workflow.md`, `manifest.yaml`, and `supervisor_checks.yaml` have been removed; they were documentation-only and not used at runtime.

| Path | Purpose | Example |
|------|------|------|
| `install.sh` | Official one-click install (referenced from GitHub raw; **do not move**) | `curl …/scripts/install.sh \| bash` |
| `install-go.sh` / `ensure-go.sh` | Go toolchain install | Server bootstrap |
| `hooks/` | Git / CI hook examples | `audit-tool.example.sh` |
| `ops/deploy/` | Deploy, nginx patches, fast rollout | `deploy_agent_server.py` |
| `ops/probe/` | Integration probes, diagnostics, audits | `probe_agent_bff.py` |
| `ops/verify/` | Acceptance and E2E verification | `verify_agent_chat_e2e.py` |
| `ops/workflow/` | Manual pre-market / intraday / post-market triggers | `run_hk_intraday_live.py` |
| `ops/fix/` | One-off fixes, syncs, and migrations | `fix_llm_key.py` |
| `ops/misc/` | Other ops tools | `find_ghsemail_user.py` |
| `dev/` | Local development / bench / small Go tools | `bench_dashboard.py` |

## Conventions

- **Temporary scripts**: names starting with `_` are for local use only, already ignored in `.gitignore`; delete them when done.
- **Probe output**: do not commit files such as `probe_*_out.txt` or `*_result.json`; delete them after the run.
- **Servers**: `geegoo-agent` should stay aligned with this repo's `scripts/` (`git pull`). Scattered `_*.py` files on TradingBot/GeeGooBot are historical temp files and can be safely deleted.

## Common commands

```bash
# Intraday decision live test (HK stocks)
python scripts/ops/workflow/run_hk_intraday_live.py

# Clear all user intraday reports
python scripts/ops/workflow/clear_all_intraday_reports.py

# Agent deploy
python scripts/ops/deploy/deploy_agent_server.py
```
