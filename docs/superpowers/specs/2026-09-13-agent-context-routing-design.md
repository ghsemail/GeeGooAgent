# Agent 上下文路由设计（TurnPlan 仅观测）

> **日期:** 2026-09-13  
> **状态:** 待审阅  
> **决策:** 方案 **B** — 保留 TurnPlan/domain 分类用于观测与 eval，**不参与**工具过滤、clarify 短路、sticky 路由。

## 1. 背景与目标

### 问题

GeeGooAgent 执行层已对齐 Cursor/Codex（ReAct + Skill + 子 Agent + clarify），但在 **进入 ReAct 之前** 仍有多层 Go 逻辑替 Agent 做决定：

- `deterministicPreLLMPlan` / `sanitizeLLMPlan` / `applyStickySessionPlan` / `applyActiveTaskGuard`
- `mapClarifyChoice`（clarify 答案 token 映射）
- `tryPresetClarify`（classify 定 clarify 后直接阻塞 UI，跳过 Agent）
- `applyTurnToolSchemas` / `FilterSchemas`（按 domain 裁工具）
- `ProbeExecutionProfile` + slot（`ShouldResolveNewSymbolForProbe`）选 execution profile
- `tryExecutionProfileRetry`（验工具链并打回）

这与产品目标冲突：**一个 Agent + 上下文决定一切**；规则与约束应 **注入上下文**，而非 bypass Agent。

### 目标

| 目标 | 说明 |
|------|------|
| **单 Agent 路由** | 每轮用户消息 → 组装上下文 → ReAct；由模型选工具、clarify、delegate |
| **TurnPlan 仅观测** | classify 仍跑，写 `turn_plan` 事件、`LastTurnDomain` 等，供 Dashboard / Plan-only eval |
| **规则即上下文** | Skill、`tool_routing`、execution contract 以 fragment 注入，不强制执行 |
| **Harness 保留** | 参数 coerce、写操作 PendingPlan、clarify UI 协议、事件 emit |

### 非目标

- 不删除 `domaincatalog` 或 TurnPlan JSON schema（观测与 eval 仍需要）
- 不在本阶段重写 Workflow 引擎（盘前/盘中仍确定性）
- 不删除 eval 套件（调整断言语义，不删用例）

---

## 2. 目标架构

```text
用户消息
  │
  ├─ [观测] IntentPlanner.classify → TurnPlan（仅 telemetry + skill hint）
  │
  ├─ [上下文] Soul → AGENTS → SessionSummary → SessionTaskState → Skill(s) → ToolRouting
  │
  └─ [执行] ReAct Agent（全 chat toolset + clarify + delegate）
        │
        harness: CoerceArguments · PendingPlan · ClarifyFn · emit events
        │
        optional [eval strict]: execution_profile verify + retry（默认关闭）
```

### SessionTaskState（新 fragment）

从 **上一轮及本轮已执行 tool 结果** 解析结构化任务态，替代 slot/sticky token 推断。

```markdown
## 当前会话任务（来自工具结果，请据此理解续轮）
- task: signal_probe | backtest_run | stock_analysis | (none)
- symbol: NVDA (US) | 00700.HK | —
- strategy: SAR+MACD | RSI阈值信号 | —
- frequency: 60m | —
- months_back: 3 | —
- last_tools: search_code → probe_bot_signal_series
```

**来源（优先级）：**

1. 最近成功的 `probe_bot_signal_series` / `run_strategy_backtest` / `search_code` 参数与返回值
2. Session metadata（若 Web 左栏 sync 已写入）
3. 无数据时省略本节（不猜）

**注入位置：** `loop.go` Phase 2，`KindWorkingState` fragment，prio 介于 clock 与 procedural skill 之间。

---

## 3. TurnPlan：观测 vs 路由（边界）

### 保留（观测）

| 行为 | 说明 |
|------|------|
| `IntentPlanner.Plan()` | LLM classify + JSON 解析 |
| `emit("turn_plan", …)` | Dashboard / trace |
| `session.LastTurnDomain/Mode/Act` | 会话元数据、Plan-only eval |
| `planForDomain` → Skills 列表 | **作为 skill 加载 hint**，非强制 |
| classify prompt / domain 枚举 | 观测标签 SSOT |

### 删除或降级（路由）

| 现逻辑 | 目标行为 |
|--------|----------|
| `deterministicPreLLMPlan` | **删除**；回测动词约束写入 Skill + ToolRouting |
| `sanitizeLLMPlan` hasAny 覆盖 | **删除** |
| `applyStickySessionPlan` | **删除**；由 SessionTaskState + Agent 理解续轮 |
| `applyActiveTaskGuard` | **删除** |
| `mapClarifyChoice` 硬映射 | **删除**；clarify 答案原样 append 为 user message |
| `tryPresetClarify` 短路 ReAct | **删除**；classify 的 clarify 建议改为 **ClarifyHint fragment** |
| `applyTurnToolSchemas` / `FilterSchemas` 裁工具 | **默认 no-op**；全量 chat schemas |
| `filterExecutionProfileSchemas` | **删除** |
| `ProbeExecutionProfile` + slot | **删除** profile 选择；hint 来自 Skill |
| `tryExecutionProfileRetry` | **config 默认 off**；仅 eval strict 开启 |

### ClarifyHint fragment（替代 preset clarify）

当 classify 返回 `mode=clarify` 时，注入只读建议（Agent 可忽略）：

```markdown
## 分类器建议（仅供参考，由你决定是否 clarify）
- suggested_question: 你是想做哪一件？
- suggested_choices: 个股/指标分析 / 测买卖点 / 跑回测看收益 / 先问答，先不操作
- reason: 提到 MACD 但未说明要分析、测点还是回测
```

Agent 自行调用 `clarify` 工具；harness 仍走现有 `ClarifyFn` + Web sheet。

---

## 4. Skill 加载

### 现况

`turnPlan.Skills` 来自 `planForDomain`，且与工具 allow-list 绑定。

### 新策略

1. **主路径：** `skillLoader.Match(userText, sessionSummary)` — 按 skill metadata / 关键词（现有 procedural loader 能力扩展）
2. **辅助 hint：** classify 产出的 skills **union** 进匹配结果，不 exclusive
3. **上限：** 仍 cap 匹配 skill 数量（如 3），避免 token 爆炸

TurnPlan 的 `ToolsAllow` **不再** 传入 `FilterSchemas`。

---

## 5. Execution Profile

### 运行时

- 不再根据 userText slot 选择 `LastExecutionProfile`
- 可选：ReAct 结束后 **异步** 用 classify domain + 实际 tools 推算 profile，仅写 telemetry（`execution_profile_observed`）

### Eval strict 模式

`config.json`:

```json
{
  "agent": {
    "execution_profile_enforce": false
  }
}
```

- `false`（默认）：无 retry，无 schema 过滤
- `true`：恢复 `tryExecutionProfileRetry` + verify（CI / 回归专用）

---

## 6. 配置与 Feature Flag

| 键 | 默认 | 说明 |
|----|------|------|
| `agent.routing.mode` | `agent_context` | `legacy` 保留旧路由（应急回滚） |
| `agent.execution_profile_enforce` | `false` | strict eval |
| `agent.turn_plan.observability_only` | `true` | 与 mode 联动 |

`legacy` 模式下行为与当前 main 一致，便于 A/B 与回滚。

---

## 7. Eval 调整

### Plan-only（不变）

仍测 `IntentPlanner` 分类准确率；**不受** routing mode 影响。

### Live verify

| 检查项 | 调整 |
|--------|------|
| `intent` | 改为 **soft**：mismatch 记 warning，不 fail（或 case 级 `expect_intent: required\|optional`） |
| `execution` | 默认验 **实际 tools** + reply；profile 仅 strict 模式 |
| `require_tools` | 保留（验 Agent 行为，非验 classify） |

文档更新：`docs/engineering/turnplan-eval.md` 增加「观测 vs 路由」说明。

---

## 8. 分阶段实施

### Phase 1 — 去 bypass（核心）

1. 新增 `sessiontask.BuildFragment(session)` → `WorkingStateFragment`
2. 新增 `ClarifyHintFragment(turnPlan)`（clarify 模式时）
3. `routing.mode=agent_context`：`applyTurnToolSchemas` → passthrough；删 `tryPresetClarify` 调用
4. 删 `deterministicPreLLMPlan`、`applyStickySessionPlan`、`sanitizeLLMPlan` hasAny、`applyActiveTaskGuard`
5. 删 classify 路径上 `mapClarifyChoice` 短路（clarify 答案当普通 user text）
6. Skill 加载改 text-match + classify hint union

**验收：** 现有 agent-loop offline 卡片通过；手动场景「probe 续轮换标的」由 Agent clarify，无 preset 短路。

### Phase 2 — Profile 与 gate 降级

1. `execution_profile_enforce` 默认 false
2. 删 slot 驱动的 `ProbeExecutionProfile` 运行时选择
3. `ProfileExecutionHint` 移入 Skill 注入（如 `strategy-signal-probe`），不再绑 profileID
4. 可选 telemetry：`execution_profile_observed` post-turn

### Phase 3 — Eval & 文档

1. Live eval intent → soft / per-case
2. 更新 turnplan-eval.md、rules-prompts.md
3. 删除 `legacy` 路径 dead code（稳定 2 周后）

---

## 9. 测试策略

| 层级 | 内容 |
|------|------|
| 单元 | `sessiontask` 解析 probe/backtest/search_code JSON |
| 单元 | routing mode passthrough schemas |
| 集成 | `loop_turnplan_test` 改断言：无 preset clarify 短路 |
| 集成 | 续轮换标的、ambiguous MACD — Agent 调 clarify |
| eval | Plan-only 仍 26+ 条；Live strict 子集跑 profile |
| 手工 | Web clarify sheet + 左栏 sync |

---

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Agent 不 clarify 导致胡调工具 | Skill 强化 + SessionTaskState；eval Live 仍验 require_tools |
| Token 增加（全工具 + 多 skill） | skill cap；context compression 不变 |
| classify 与 Agent 行为不一致 | 预期内；Dashboard 标注 `observability_only` |
| 回滚 | `routing.mode=legacy` 一键恢复 |

---

## 11. 文件 touch list（预估）

| 操作 | 路径 |
|------|------|
| 新增 | `internal/sessiontask/state.go`, `state_test.go` |
| 改 | `internal/agent/loop.go` |
| 改 | `internal/agent/loop_turnplan.go` |
| 删/改 | `internal/agent/loop_clarify.go`（preset 路径） |
| 改 | `internal/agent/loop_execution_gate.go` |
| 改 | `internal/cognition/llm_planner.go` |
| 删/缩 | `internal/cognition/intent_planner.go`（token 函数） |
| 删/缩 | `internal/slots/probe_symbol.go` |
| 改 | `internal/config/*.go`（routing flags） |
| 改 | `internal/eval/*`（soft intent） |
| 改 | `docs/engineering/turnplan-eval.md` |

---

## 12. 成功标准

1. `routing.mode=agent_context` 下，无 Go 路径在 ReAct 前调用 clarify 或裁工具
2. 「换一个标的」类续轮：Agent 发起 clarify（带 choices）→ probe，不依赖 slot
3. Plan-only eval 通过率不低于迁移前
4. Live eval 在 soft intent 下 require_tools 通过率不低于迁移前（允许 ±5% 波动观察期）
5. Dashboard 仍可看 `turn_plan.domain` 分布

---

## 13. 审阅确认项

- [ ] Phase 1 范围是否 OK（先不去掉 classify LLM 调用）
- [ ] Live eval intent 改 soft 是否可接受
- [ ] `legacy` 回滚 flag 保留 2 周是否足够
