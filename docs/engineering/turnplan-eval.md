# TurnPlan Eval

TurnPlan 评测验证 Agent 每轮用户输入的 **意图路由**、**ReAct 工具调用** 与 **助手回复质量**。

## 两种模式

| 模式 | 入口 | 耗时 | 说明 |
|------|------|------|------|
| **Plan-only** | `POST /v1/dashboard/eval/run-turn-plan` | 秒级 | 只测 LLM/规则分类，不跑 Chat |
| **Live** | Dock Chat → `POST .../cases/{id}/verify` | 分钟级 | 真实 SSE 对话 + 结构化校验 + LLM judge |

Live 用例 **不能** 直接 `POST .../cases/{id}/run`（会 400）；必须先完成 `POST /v1/chat/stream`，再带 `session_id` 调 verify。

## 用例结构（23 条 Live）

源码：`internal/eval/turnplan_cases.go` → `IndividualTurnPlanEvalCases()`。

| 分类 ID | 标题 | 条数 | 代表用例 |
|---------|------|------|----------|
| `stock_analysis` | 股票分析 | 4 | 查股价、技术面续问、切换标的、代词指代 |
| `signal` | 信号 / 策略 | 3 | 列策略、列策略后 probe、直接 probe |
| `backtest` | 策略回测 | 5 | 显式/口语回测、分析后回测、DCA 回测 |
| `clarify` | 灰区 / 澄清 | 2 | 模糊 MACD、分析+回测复合句 |
| `chat` | 闲聊 / QA | 2 | 指标释义、测点后问信号质量 |
| `bot_manage` | Bot 管理 | 3 | Reminder / Grid / SmartTrade |
| `history_report` | 历史 / 报告 | 2 | 回测历史、盘前报告 |
| `knowledge_news` | 知识 / 新闻 | 2 | 知识库、财经新闻 |

每条 Live 用例：

- **独立 session**（`session_cleanup: before_run`）
- **多轮**：前置轮次在 `setup_messages`，最后一轮带 `judge: true`
- **expect_sop: false**（统一走 plan + ReAct，无确定性 SOP 短路）
- **expect_reply** + **LLM judge**（`internal/eval/turnplan_expect_reply.go`）

Plan-only 套件：`DefaultTurnPlanSuite()`，用 `LastDomain` 模拟多轮上下文，不占 Chat session。

## Verify 检查项

Live verify（`VerifyTurnPlanLiveFull`）依次检查：

1. **routing** — `last_turn_plan.domain/mode/sop` 与期望一致
2. **tools** — `require_tools` / `forbid_tools`
3. **reply_length** — 最短字符数
4. **must_cover** — 关键词命中
5. **llm_judge** — 辅助模型按 rubric 打分（默认 ≥ 0.7）

## 数据与 API

- Dashboard 用例表：`agent_eval_cases`（`options_json.category = turn_plan`）
- Seed SQL：`internal/infra/schema.sql`（SQLite）、`internal/infra/pgschema/postgres_eval.sql`（Postgres）
- 重新生成 seed：`go run scripts/gen_turnplan_eval_sql.go`
- 生成自动化 manifest：`go run scripts/eval/gen_turnplan_cases_json.go`

### 主要 HTTP 路由

```
GET  /v1/dashboard/eval/cases
POST /v1/dashboard/eval/run-turn-plan          # plan-only 批量
POST /v1/dashboard/eval/cases/{id}/verify      # live verify（需 session_id）
POST /v1/chat/stream                           # live 对话
```

## 自动化脚本

见 `scripts/eval/README.md`。

推荐流程：

1. 修改 Go 用例 → 跑 `gen_turnplan_eval_sql.go` + `gen_turnplan_cases_json.go`
2. commit + push → 部署 agent-runtime
3. `python scripts/eval/migrate_turnplan_eval_db.py` 同步 PG
4. `python scripts/eval/run_turnplan_live_remote.py --category stock_analysis` 冒烟

## 设计原则（当前版本）

- **LLM-first 路由**：`LLMPlanner` 主判，`RulePlanner` 仅 hint/fallback
- **无 SOP 短路**：`ShouldRunDomainSOP()` 恒 false，Loop 不 `TryRunFromPlan`
- **口语化对话**：eval 用户话术完整自然，避免「可以」「这边呢」等半句话
