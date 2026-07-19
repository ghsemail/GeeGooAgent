# 业务能力分期

> **实现细节与缺口清单** → [implementation-status.md](../implementation-status.md)（优先读）。  
> **平台内核交付（P1–P8）** → [deploy/hermes-parity-roadmap.md](../../../deploy/hermes-parity-roadmap.md)。  
> **Agent Runtime 控制面改造**（与业务 Phase 正交）→ [agent-runtime-migration-plan.md](../agent-runtime-migration-plan.md)；定稿 → [agent-runtime-architecture.md](../agent-runtime-architecture.md)。

## Phase 路线图

| Phase | 主题 | 状态 | 交付物 |
|-------|------|------|--------|
| 0 | 平台内核 | ✅ | Agent、Tool、SQLite、Workflow 框架 |
| 1 | 盘前 MVP | ✅ | `geegoo run pre_market` 端到端 |
| 2 | 盘后 | 📋 | `post_market` 已注册，无步骤与资源 |
| 3 | 盘中 | 📋 | `intraday` 已注册，无步骤；富途 Tool ⚠️ |
| 4 | Chat 按需分析 | ⚠️ | `geegoo chat` + `market` toolset |
| 5 | 策略 | ⚠️ | `strategy` toolset；后端简化 |
| 6 | Bot 管理 | ⚠️ | CRUD Tool；ApprovalGate |
| 7 | Prompt 模板 | ⚠️ | 六个 CRUD Tool 已注册 |

## 当前优先缺口

| 项 | 状态 | 说明 |
|----|------|------|
| `post_market` / `intraday` workflow | 📋 | 需 `skills/*` + `workflow/*.go` 步骤 |
| `fetch_*_news` | ❌ | 无 script runner |
| `recall_yesterday_summary` | ❌ | Episodic 未做 |
| 富途 `get_position` 等 | ⚠️ | 依赖 MCP 配置 |
| Bot 侧 scheduler | ❌ | Agent 不自动下单 |

Tool 明细 → [layers/L2-tools/tools-status.md](../layers/L2-tools/tools-status.md)

## 原则

- Phase 1（盘前）为不可退让的 MVP 基线
- 基础设施先于业务自动化（SQLite、Checkpoint 先于 intraday）
- Bot 写操作必须 ApprovalGate；scheduler 不触发交易

## 依赖

```text
Phase 0 → Phase 1（必达）
Phase 1 → Phase 2 / 3（可并行）
Phase 4+ 依赖 Runtime + L2 Clients 稳定
```

历史任务清单（归档）→ [archive/phases-roadmap.md](../../archive/phases-roadmap.md)
