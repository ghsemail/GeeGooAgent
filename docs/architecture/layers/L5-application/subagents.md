# L5 — Subagent 委派

## 状态：✅ 已实现（`delegate_task`）

`internal/agent/subagent.go` 提供独立子 Agent 回合；禁止嵌套 delegate。规划中的按角色 tool 子集（NewsCollector 等）仍为可选增强，见下表。

| 名称 | 工具子集 | 说明 |
|------|----------|------|
| NewsCollector | fetch_*_news | 依赖新闻 script runner |
| StockAnalyst | analysis + report | 可由 workflow 步骤替代 |
| StrategyAdvisor | strategy + loopback | chat toolset 已覆盖 |
| BotManager | bot + reminder | chat toolset 已覆盖 |

若实现：子 Session `max_steps` 限制、禁止嵌套 spawn、只向父 Session 返回摘要。
