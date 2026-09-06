# TurnPlan Eval

TurnPlan 评测覆盖 **路由（domain/mode）**、**工具调用**、**回复语义（LLM judge）**。完整说明见 [docs/engineering/turnplan-eval.md](../../docs/engineering/turnplan-eval.md)。

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

# 部署后同步 eval 用例到 PG
python scripts/eval/migrate_turnplan_eval_db.py
```

## 源码位置

- 用例定义：`internal/eval/turnplan_cases.go`
- 语义 rubric：`internal/eval/turnplan_expect_reply.go`
- Verify 逻辑：`internal/eval/turnplan_live.go`
- HTTP API：`internal/runtimeapi/dashboard_eval_turnplan.go`
