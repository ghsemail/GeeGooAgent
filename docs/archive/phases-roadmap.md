# 分期路线图

## Phase 0 — 平台内核

| 层   | 交付                                                                                |
| --- | --------------------------------------------------------------------------------- |
| L0  | EventBus, StateStore, Checkpoint, Scheduler + Logging/Tracing/Secrets/Env/Sandbox |
| L1  | Model Gateway + Cost stub                                                         |
| L4  | Planner, Executor, StateMachine, WorkflowEngine, ReAct Loop                       |
| L3  | Session/Working Memory + Compaction L1-L2                                         |
| L2  | ToolRegistry 框架 + Clients 骨架                                                      |
| L5  | SkillLoader + pre_market stub                                                     |

**不含**盘前业务端到端。

## Phase 1 — MVP 盘前

- 全部盘前 Tool
- `skills/pre_market` 完整
- Supervisor + systemd 08:00
- 替代 Hermes cron

## Phase 2-3

- post_market / intraday
- StockAnalyst 默认、webhook

## Phase 4-6

- geegoo skill：chat、strategy、bot_manager

## 成功标准（Phase 1）

1. 交易日 8:00 自动触发
2. 每股 md + API 入库，字段完整
3. execution-log 真实时间戳
4. Supervisor 漏步可 resume
5. 崩溃可 `geegoo-agent resume`
6. 产出与 Hermes 可比（抽检 3 股）

## 工作量估算

| Phase       | 行数约      |
| ----------- | -------- |
| 0-1 MVP     | ~2700    |
| 0-3 全工作流    | ~4100    |
| 4-6 geegoo 全量 | ~6600 合计 |
