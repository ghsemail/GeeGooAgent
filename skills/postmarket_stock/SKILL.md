---
name: postmarket_stock
description: Post-market session summary reports on trading days.
version: "1.0.0"
---

# postmarket_stock Skill Pack

Scheduled post-market workflow per geegoo `post-market-workflow.md`. `session_bias` and `vs_stock_premarket` are computed in Go, not by LLM.

## Assets

| File | Description |
| --- | --- |
| `manifest.yaml` | Tool allowlist and steps |
| `workflow.md` | Business workflow |
| `template.md` | Markdown template |

## Run

```bash
geegoo run postmarket_stock --config config.json
geegoo run postmarket_stock --dry-run
```
