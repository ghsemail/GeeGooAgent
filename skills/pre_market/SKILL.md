---
name: pre_market
description: Generate pre-market analysis reports on trading days.
version: "2.0.0"
---

# pre_market Skill Pack

GeeGoo pre-market automation is executed by the Go `WorkflowRunner` according to `manifest.yaml`. The LLM is used only for analysis and report synthesis; API calls, checkpointing, and persistence are handled by the Go runtime.

## Assets

| File | Description |
| --- | --- |
| `manifest.yaml` | Tool allowlist and workflow step definitions. |
| `workflow.md` | Full business workflow. |
| `template.md` | Markdown report template. |

## Related Rules

- `rules/api-routing.md` - 3xxx GeeGoo service routing.
- `rules/attitude-mapping.md` - attitude to result mapping.
- `rules/report-format.md` - required `createPreMarketReport` fields.

## Run

```bash
geegoo run pre_market --config config.json
geegoo run pre_market --dry-run --config config.json
geegoo resume --session <id> --config config.json
geegoo chat --config config.json
```

## Non-Trading Days

When `check_trading_day` returns `false`, the workflow finishes immediately and does not create per-stock reports.
