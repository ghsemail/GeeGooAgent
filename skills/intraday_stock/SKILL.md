---
name: intraday_stock
description: Intraday trade decision on bot/reminder signals.
version: "1.0.0"
---

# intraday_stock Skill Pack

Signal-triggered intraday decision workflow. Go `WorkflowRunner` executes deterministic steps; `result`/`confidence` are rule-based per geegoo `intraday-workflow.md`.

## Assets

| File | Description |
| --- | --- |
| `manifest.yaml` | Tool allowlist and step definitions |
| `workflow.md` | Business workflow |
| `template.md` | Markdown report template |

## Run

```bash
geegoo run intraday_stock --code 00700.HK --stock-name 腾讯控股 \
  --bot-id <id> --bot-name my-bot --bot-type GRID \
  --frequency 5m --trade-type 信号买入
geegoo run intraday_stock --dry-run
```

Default dry-run uses sample `00700.HK` / `dry-run-bot-1`.
