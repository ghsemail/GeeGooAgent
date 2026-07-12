# 可观测性

## 产物一览

| 产物               | 路径             | 模块             |
| ---------------- | -------------- | -------------- |
| execution-log.md | `{date}/`      | Logging + Tool |
| session JSON     | `sessions/`    | StateStore     |
| checkpoint       | `checkpoints/` | Checkpoint     |
| metrics.json     | `{date}/`      | Cost + Tracing |
| journald         | systemd        | Logging        |

## 排障路径

1. `journalctl -u geegoo-agent` — 结构化事件
2. `execution-log.md` — 业务步骤
3. `geegoo-agent session show <id>` — step_records
4. `metrics.json` — token/费用/延迟

## 告警（后期）

- Supervisor 硬失败 → 飞书 webhook
- 日 LLM 费用超阈值 → CostManager

## MVP

execution-log + session + metrics + journald。