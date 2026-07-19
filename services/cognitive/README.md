# Cognitive Advisor (optional sidecar)

Stateless Python HTTP service for **suggestion-only** cognition (ranking / evaluation).  
Go Agent Kernel remains SSOT; this process must never own loop, tools, or session writes.

## Run locally

```bash
python services/cognitive/advisor_server.py
# default http://127.0.0.1:3410/health
```

## Enable in GeeGooAgent

`config.json`:

```json
{
  "advisor": {
    "enabled": true,
    "base_url": "http://127.0.0.1:3410",
    "timeout_sec": 3,
    "ranker": true,
    "evaluator": true
  }
}
```

Default: **disabled** — behavior identical to pure Go cognition.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/health` | Liveness |
| POST | `/v1/advisor/rank` | `{"items":[...]}` → `{"items":[...]}` |
| POST | `/v1/advisor/evaluate` | turn snapshot → `accept` / `retry_suggested` |

Forbidden in responses: `tool_calls`, `state`, `workflow_decision`, etc. (enforced by Go client).

## Degradation

If the sidecar is down or returns errors, Go falls back to `IdentityRanker` / `AcceptAllEvaluator`; chat continues.

## Production (optional)

Enable in `config.json` (`advisor.enabled: true`) only when the sidecar is running.

```bash
# systemd (adjust User/WorkingDirectory to match server layout)
sudo cp deploy/systemd/geegoo-advisor.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now geegoo-advisor
curl -s http://127.0.0.1:3410/health
```

GeeGooAgent main service does **not** depend on advisor health; advisor failure is non-fatal.
