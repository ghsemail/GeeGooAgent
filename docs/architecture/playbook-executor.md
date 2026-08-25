# Playbook 确定性执行器

## 背景

Chat 原先流程：keyword 匹配 playbook → 注入 SKILL.md 文本 → LLM 自由选 tool。  
Playbook 与 ToolRouting 都是 **prompt 软约束**，无法保证 `strategy-backtest-run` 一定调用 `run_strategy_backtest`。

Web「策略回测」页不走 Agent，而是前端固定 pipeline（probe → 本地 simulate → 落库）。

## 目标

**长期路径**：playbook 命中后进入 **确定性执行模式**：

1. **Plan**：LLM 或规则解析用户意图 → 结构化 JSON（标的、信号、周期、资金）
2. **Execute**：固定步骤顺序调用 tool（与 playbook SOP 一致）
3. **Reply**：模板或 LLM 格式化结果

Web Chat 与飞书 Chat **共用**同一 Agent loop，因此两端行为一致。

## 架构

```text
RunTurn
  ├─ procedural memory (keyword → 注入 playbook 文本，仍保留)
  ├─ playbookexec.Router.TryRun  ← 新增
  │     ├─ Route(matchedSkills, userText)
  │     ├─ buildBacktestPlan (heuristic / LLM JSON)
  │     └─ 固定步骤: search_code → get_signal_* → run_strategy_backtest
  └─ ReAct loop (未命中 executor 时 fallback)
```

## 当前覆盖

| Playbook | Executor | 固定 Tool 链 |
|----------|----------|--------------|
| `strategy-backtest-run` | ✅ | `search_code` → `get_signal_combinations` / `get_index_signals` → `run_strategy_backtest` |

触发条件：

- 用户话含回测意图（`BacktestRunIntent`）
- 匹配 skill 含 `strategy-backtest-run` 或 `strategy-backtest`
- **不**含 explicit DCA/网格/loopback（`BacktestDCABypass`）

## 与 Workflow Skill 的关系

| 模式 | 入口 | 用途 |
|------|------|------|
| **playbookexec** | Chat（Web/飞书） | 对话式回测，可 clarify |
| **workflow.Run** | `geegoo run` / scheduler | 批量、resume、Supervisor（后续可注册 `strategy_backtest_run` workflow） |

## 扩展清单

1. 新 playbook：在 `playbookexec.Router.TryRun` 增加 case + 实现 `runXxx`
2. Tool schema 收窄：executor 内只调用白名单 tool，不暴露给 LLM
3. Workflow 注册：同一套步骤表复用到 `internal/workflow/backtest/`
4. Supervisor：对 executor 结果做 pass/recoverable 验收

## 相关文件

- `internal/playbookexec/` — Router、plan、backtest 执行
- `internal/agent/loop.go` — TryRun 钩子（ReAct 之前）
- `internal/tools.Context.FullCatalogPayload` — executor 拉全量 buy_signal
