# L5 — Subagent 委派

## 状态：❌ 未实现

当前 Orchestrator **直接**调用 Tool Registry；无 `spawn_subagent`、无独立子 Session 包。

## 规划角色（未编码）

| 名称 | 工具子集 | 说明 |
|------|----------|------|
| NewsCollector | fetch_*_news | 依赖新闻 script runner |
| StockAnalyst | analysis + report | 可由 workflow 步骤替代 |
| StrategyAdvisor | strategy + loopback | chat toolset 已覆盖 |
| BotManager | bot + reminder | chat toolset 已覆盖 |

若实现：子 Session `max_steps` 限制、禁止嵌套 spawn、只向父 Session 返回摘要。
