# TurnPlan Eval

TurnPlan 评测覆盖 **路由（domain/mode）**、**工具调用**、**回复语义（LLM judge）**。

- 通用机制（给其他 Agent 复制）：[docs/architecture/platform-blueprint/eval-build-guide.md](../../docs/architecture/platform-blueprint/eval-build-guide.md)
- 本仓库套件与操作：[docs/engineering/turnplan-eval.md](../../docs/engineering/turnplan-eval.md)

## 目录

| 文件 | 用途 |
|------|------|
| `turnplan_cases.json` | Live 用例清单（由 Go 生成，勿手改） |
| `gen_turnplan_cases_json.go` | 从 `internal/eval` 生成 JSON |
| `run_turnplan_live.py` | 在 agent 主机上跑 live eval |
| `run_turnplan_live_remote.py` | 本机 SSH 触发远程跑 eval |
| `migrate_turnplan_eval_db.py` | 将 eval seed 同步到 Postgres |

## 常用命令

```bash
# 改 internal/eval/turnplan_cases.go 后
go run scripts/gen_turnplan_eval_sql.go          # 更新 schema.sql / postgres_eval.sql
go run scripts/eval/gen_turnplan_cases_json.go   # 更新 turnplan_cases.json

# Plan-only 规则回归（无 Chat，毫秒级）
go test ./internal/eval/... -run TestTurnPlan
curl -X POST http://127.0.0.1:3400/v1/dashboard/eval/run-turn-plan

# Live eval（必须先 chat，再 verify）
python scripts/eval/run_turnplan_live_remote.py --list
python scripts/eval/run_turnplan_live_remote.py --category stock_analysis
python scripts/eval/run_turnplan_live_remote.py --case turn_plan_stock_price

# Dashboard 自动测评（runtime 本机）
# 打开 GET /v1/dashboard/eval/auto
# 或按分类后台跑：
curl -X POST http://127.0.0.1:3400/v1/dashboard/eval/jobs \
  -H "Authorization: Bearer $GEEGOO_AGENT_RUNTIME_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title":"股票分析冒烟","category_ids":["stock_analysis"]}'


# 部署后同步 eval 用例到 PG
python scripts/eval/migrate_turnplan_eval_db.py

# 本机 SQLite（默认 ~/.geegoo/data/geegoo.db）同步 TurnPlan + strategy_backtest + workflow
go run scripts/eval/sync_eval_cases_local.go
go run scripts/eval/sync_eval_cases_local.go -db D:/path/to/geegoo.db

# Workflow eval（PG seed + live smoke）
python scripts/eval/migrate_workflow_eval_db.py
python scripts/eval/run_workflow_generate_cognition_random_remote.py
python scripts/eval/run_workflow_signal_diagnose_remote.py
```

## Workflow eval 与 Chat Loop（#106）

PR #106 后默认 Chat Loop 为 `skill_injection=catalog`、`chat_open_flow_tool` 关闭。**#106 回归绿线**建议只跑默认路径用例 `workflow_signal_diagnose_sar_tencent`（`run_workflow_signal_diagnose_live.py` / remote 包装）。

| 用例 | 说明 |
|------|------|
| `workflow_signal_diagnose_sar_tencent` | 默认绿线：以诊断回复长度 + 关键词为准；`workflow_card` / `skill_view` / Episode 均为可选观察，live 脚本对缺 card 仅 WARN |
| `workflow_generate_strategy_cognition` 等标题含 **legacy open-flow** | 仍假设旧 `run_chat_workflow_turn` / Workflow Runner；全量 Workflow 自动测评在默认 Loop 下可能假红，跑前需临时开 open-flow 或排除这些 case |

**Verify 语义**：Dashboard `pass_keywords` 为命中**任一**即过（OR）；`run_workflow_signal_diagnose_live.py` 冒烟对 SAR·腾讯 用 **AND**，且要求诊断/命中/触发至少其一 — 以 live 脚本为准。

## 源码位置

- 用例定义：`internal/eval/turnplan_cases.go`
- 语义 rubric：`internal/eval/turnplan_expect_reply.go`
- Verify 逻辑：`internal/eval/turnplan_live.go`
- HTTP API：`internal/runtimeapi/dashboard_eval_turnplan.go`
- 自动测评 Job：`internal/runtimeapi/dashboard_eval_auto.go`
- 运营后台 Tab：`trading_operation` → Eval → 自动测评
