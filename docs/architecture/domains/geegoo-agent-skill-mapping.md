# geegoo skill 迁移映射

| 原资产                                 | 目标                              |
| ----------------------------------- | ------------------------------- |
| `references/pre-market-workflow.md` | `skills/pre_market/workflow.md` |
| `references/pre-market-template.md` | `skills/pre_market/template.md` |
| `references/post-market-*`          | `skills/post_market/`           |
| `references/intraday-*`             | `skills/intraday/`              |
| `skills/finance-news/` 等            | `skills/bundled/`               |
| `cron/*.json`                       | **废弃** → systemd timer          |
| `SKILL.md` 经验                       | `rules/`                        |
| `docs/geegoo-mcp/`（SSOT 镜像） | `docs/reference/geegoo-mcp/` + `domains/` |

## MVP

仅迁移 pre_market 相关。