# Phases — 分期交付

从 Phase 0 基础设施到 Phase 7 高级 Prompt 管理的**实现路线图**。

## 模块设计说明

`phases/` 定义「**先做什么、后做什么**」，防止 MVP 膨胀。架构六层文档描述**终态**；本目录描述**到达路径**——哪一 Phase 解锁哪些 Tool、哪些 Skill、哪些 L0 模块。

**分期哲学**

| 原则 | 说明 |
|------|------|
| MVP 不变 | Phase 1 仅 `pre_market` 盘前端到端，替代 Hermes cron |
| 基础设施先行 | Phase 0：L0 四件套 + L4 最小 Loop + L1 Gateway 轻量版 |
| 能力按风险递增 | Bot CRUD（Phase 6）需 `wait_for_human`；Scheduled 永不加载 |
| 文档与代码同 Phase | 某 Phase 标记「完成」= 对应 Tool 可测 + deploy 可跑 |

**Phase 一览**

| Phase | 主题 | 主要交付 | 详见 |
|-------|------|----------|------|
| 0 | 地基 | EventBus、StateStore、Checkpoint、Scheduler、Gateway stub | [roadmap.md](./roadmap.md) |
| 1 | 盘前 MVP | pre_market Skill、19 Tool、双 Client、本地报告 | roadmap |
| 2 | 盘后 | post_market、get_bot_log、create_post | roadmap |
| 3 | 盘中 | intraday、get_ticker/get_broker、持仓 | roadmap |
| 4 | 按需分析 | on_demand_analysis、search_code | roadmap |
| 5 | 策略 | 信号、generate_*、loopback | roadmap |
| 6 | Bot 管理 | 全量 Bot/Reminder CRUD、switch_bot | roadmap |
| 7 | Prompt 高级 | 竞品/ETF 模板 CRUD | roadmap |

**依赖关系**

```text
Phase 0 ──▶ Phase 1（MVP 必达）
Phase 1 ──▶ Phase 2 / 3（可并行）
Phase 4+ ──▶ 依赖 Phase 1 的 Runtime + Clients 稳定
Phase 6 ──▶ 依赖 Sandbox + wait_for_human + geegoo-skill-mapping 完整
```

**与 Cursor Plan 的关系**

Cursor Plan 文件仅作任务勾选；**分期边界以 [roadmap.md](./roadmap.md) 与本 README 为准**。若 Plan 与 roadmap 冲突，以本目录为准。

## 文档索引

- [roadmap.md](./roadmap.md) — 详细任务清单与验收标准
