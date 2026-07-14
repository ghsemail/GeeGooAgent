# Phases — 分期交付与当前状态

架构文档描述**终态**；本目录描述**到达路径**与**当前完成度**。

## Hermes 对齐（P1–P8）✅ 已完成

详见 [../../deploy/hermes-parity-roadmap.md](../../deploy/hermes-parity-roadmap.md)：

| 阶段 | 交付 |
|------|------|
| P1–P3 | Agent 核心、Tool 注册、MCP 客户端 |
| P4–P5 | SQLite、Evidence、Workflow、Supervisor |
| P6–P7 | Chat UI、压缩、Scheduler |
| P8 | Cutover verify、文档 |

## 业务能力 Phase（路线图）

| Phase | 主题 | 状态 | 主要交付 |
|-------|------|------|----------|
| 0 | 地基 | ✅ | infra、Gateway、最小 Loop |
| 1 | 盘前 MVP | ✅ | pre_market、workflow、~19 Tool 路径 |
| 2 | 盘后 | 📋 | post_market Skill 占位 |
| 3 | 盘中 | 📋 | intraday、富途三接口 |
| 4 | 按需分析 | ⚠️ | chat + market toolset（无独立 Skill pack） |
| 5 | 策略 | ⚠️ | 82 Tool 含 strategy toolset；Analyze/Go 简化版 |
| 6 | Bot 管理 | ⚠️ | CRUD 可用；无 switch_bot / wait_for_human / scheduler |
| 7 | Prompt 高级 | ⚠️ | 模板 CRUD 已注册；catalog-api 原生 |

## 当前缺口（对话相关）

| 能力 | 状态 |
|------|------|
| `switch_bot` | ❌ |
| `wait_for_human` | ❌ |
| `fetch_*_news` script runner | ⚠️ skipped |
| `get_ticker` / `get_position` | ⚠️ Noop |
| intraday / post_market workflow | 📋 占位 |

完整 Tool 状态 → [../reference/geegoo-agent-tools-tree.md](../reference/geegoo-agent-tools-tree.md)

## 分期哲学

| 原则 | 说明 |
|------|------|
| MVP 不变 | Phase 1 = pre_market 端到端 |
| 基础设施先行 | SQLite + Workflow 先于 Bot 全自动 |
| 风险递增 | Bot 写操作需审批门控；scheduler 不自动交易 |
| 文档随代码 | 架构 README 与 tools-tree 同步更新 |

## 依赖关系

```text
Phase 0 ──▶ Phase 1（必达）
Phase 1 ──▶ Phase 2 / 3（可并行）
Phase 4+ ──▶ 依赖 Runtime + Clients 稳定
Phase 6 ──▶ wait_for_human + switch_bot + Bot scheduler
```

## 文档索引

- [roadmap.md](./roadmap.md) — 历史任务清单
- [../README.md](../README.md) — 架构总索引
- [../phases/README.md](./README.md) — 本文件

## 与 Cursor Plan 的关系

Cursor Plan 仅作任务勾选；**分期边界以本目录 + roadmap 为准**。
