---
name: post_market
description: Post-market session summary reports on trading days.
version: "1.0.0"
---

# post_market Skill Pack

Scheduled post-market workflow per geegoo `post-market-workflow.md`. `session_bias` and `vs_pre_market` are computed in Go, not by LLM.

## Assets

| File | Description |
| --- | --- |
| `manifest.yaml` | Tool allowlist and steps |
| `workflow.md` | Business workflow |
| `template.md` | Markdown template |

## Run

```bash
geegoo run post_market --config config.json
geegoo run post_market --dry-run
```
