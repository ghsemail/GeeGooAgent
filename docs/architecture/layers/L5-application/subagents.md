# L5 — Subagent 委派

## 状态：✅ 已实现（`delegate_task` + `delegate_tasks`）

> 更新：2026-07-19。已部署至 119.45.16.112（`225615ed`）。

| 工具 | 说明 |
|------|------|
| `delegate_task` | 单子任务；独立子 Agent 回合；不写入主会话历史 |
| `delegate_tasks` | 批量 1–8 项并行；受 `delegate_max_parallel`（默认 3，最大 8）限制 |

`internal/agent/subagent.go` 提供独立子 Agent 回合；禁止嵌套 delegate（`DelegateDepth >= 1` 拒绝）。单轮多个 `delegate_task` 亦受同一 semaphore 约束。

规划中的按角色 tool 子集（NewsCollector 等）仍为可选增强，见下表。

| 名称 | 工具子集 | 说明 |
|------|----------|------|
| NewsCollector | fetch_*_news | 依赖新闻 script runner |
| StockAnalyst | analysis + report | 可由 workflow 步骤替代 |
| StrategyAdvisor | strategy + loopback | chat toolset 已覆盖 |
| BotManager | bot + reminder | chat toolset 已覆盖 |

若实现：子 Session `max_steps` 限制、禁止嵌套 spawn、只向父 Session 返回摘要。

## 验收

```bash
geegoo verify agent-loop    # delegate_task / delegate_tasks / nesting guard
geegoo inspect --quick      # 内嵌 parity 卡片
```
