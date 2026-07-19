# Agent Loop：GeeGooAgent × Hermes Agent

> 更新：2026-07-20。GeeGoo 以 `internal/agent` 与 [agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md) 为准；Hermes 以 [官方架构](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture) 与 [hermes-parity-comparison.md](../../../deploy/hermes-parity-comparison.md) 为准。  
> **线上**：119.45.16.112 基线 `20471ef`（2026-07-20）；`geegoo verify agent-loop --offline` 12/12 PASS。

## 文档概述

本文档从 **Agent Loop 机制** 角度对比 GeeGooAgent 与 Hermes Agent：ReAct 生命周期、压缩/缓存、子 Agent、审批与可观测性。不覆盖 IM 网关、Grok 式编码工具等外围能力。读完可判断「从 Hermes cron 迁到 GeeGoo 时 loop 还差什么」；实现细节与验收见架构 [agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md)、[agent-loop-verification.md](../../architecture/layers/L4-runtime/agent-loop-verification.md)。

## 摘要

| 维度 | 结论 |
|------|------|
| **循环骨架** | 两者均为标准 ReAct：`LLM(+tools) → tool_calls? → 执行 → 写回 session → 重复` |
| **核心对齐度** | GeeGoo **已对齐** Hermes agent-loop 主路径（压缩、缓存、delegate、clarify、审批、plan_gate、预算耗尽、可中断、NDJSON） |
| **最大结构差异** | GeeGoo 另有 **确定性 Workflow**（`geegoo run` / scheduler）；Hermes cron 多为「新建 Agent + skill prompt」 |
| **GeeGoo 更强** | Workflow Supervisor、Evidence、报告合成、`geegoo verify agent-loop --offline`、Cognition 注入（Evaluator 可选重试） |
| **Hermes 更强** | 18+ Provider / 3 种 API mode、Context Engine 插件、70+ 工具含终端/浏览器、Gateway/ACP、轨迹导出 |

---

## 1. 架构对照

### 1.1 入口与核心

```text
Hermes                              GeeGooAgent
────────                            ───────────
cli.py ──────────┐                  geegoo chat ─────────┐
gateway (20 IM) ─┼→ AIAgent        agent-runtime HTTP ───┼→ Agent.Run → Loop.RunTurn
cron / ACP ──────┘   run_agent.py   geegoo run / scheduler ┘→ App.RunSkill → workflow.Runner
```

Hermes 强调 **单一 `AIAgent`** 服务 CLI、Gateway、Cron、ACP。GeeGoo **平台差异在入口**（[entrypoints.md](../../architecture/entrypoints.md)），盘前从 ReAct 拆为 Workflow。

### 1.2 代码映射

| Hermes (Python) | GeeGooAgent (Go) | Loop 相关 |
|-----------------|------------------|-----------|
| `run_agent.py` — `AIAgent` | `internal/agent/agent.go` — `Agent.Run` | 统一门面 |
| `AIAgent.run_conversation()` | `internal/agent/loop.go` — `Loop.RunTurn` | 主循环 |
| `model_tools.py` | `internal/agent/tool_exec.go` + `internal/tools/registry.go` | 工具派发 |
| `agent/prompt_builder.py` | `internal/chatprompt/builder.go` | 稳定 system |
| `agent/context_compressor.py` | `internal/prompt/compressor.go` | 四阶段压缩 |
| `agent/prompt_caching.py` | `internal/llm/cache.go` | 缓存断点 |
| `tools/delegate_tool.py` | `internal/agent/delegate_tool.go` | 子 Agent |
| `tools/clarify_tool.py` | `internal/tools/clarify.go` | 用户澄清 |
| `tools/approval.py` | `internal/tools/approval.go` + `plan_gate` | 写操作审批 |
| — | `internal/cognition/` + `SetCognition` | Ranker / Evaluator / PlanPolicy |
| — | `internal/workflow/supervisor.go` | **GeeGoo 特有**（workflow） |

---

## 2. 单次 Turn 生命周期

### 2.1 共同流程

```text
用户输入 → 追加 user message → [可选] hygiene 压缩（~85%）
  └─ for round in 1..max_steps:
        → [可选] 轮前压缩（~50%）→ SanitizeMessages → Gateway.Chat
        → 无 tool_calls → 返回 assistant
        → 有 tool_calls → ExecuteBatch → 追加 tool messages → 继续
  → 达 max_steps → finishBudgetExhausted（无 tool 终局 LLM）
```

GeeGoo：`Loop.RunTurn`（`loop.go`）、`runRound`（`loop_round.go`）。流程图见 [agent-loop.md §3](../../architecture/layers/L4-runtime/agent-loop.md#3-执行流程)。

### 2.2 逐步对比

| 阶段 | Hermes | GeeGooAgent | 评价 |
|------|--------|-------------|------|
| System prompt 稳定 | ✅ | ✅ `chatprompt` | ✅ |
| Hygiene / 轮前压缩 | ✅ | ✅ 85% / 50% | ✅ |
| 流式 + thinking / tool_gen | ✅ | ✅ `stream_delta` 等 | ✅ |
| 并行 tool | ✅ | ✅ `tool_max_parallel` | ✅ |
| 中断 | ✅ | ✅ `context.Context` | ✅ |
| 预算耗尽总结 | ✅ | ✅ `finishBudgetExhausted` | ✅ |
| **Plan 门控（mutating）** | ⚠️ approval | ✅ `plan_gate` + `PendingPlan` | ✅ GeeGoo 显式挂起 |
| **并行子 Agent** | ⚠️ 有限 | ✅ `delegate_tasks` + `delegate_max_parallel` | ⚠️ GeeGoo 已支持批量并行 |
| **Evaluator 质量重试** | — | ⚠️ `eval_max_retries`（0–1，可选 Advisor） | GeeGoo 增量 |

### 2.3 GeeGoo 独有（对话 turn 内）

| 机制 | 说明 |
|------|------|
| Chat 工具拦截 | workflow 独占 tool 剔除 + `tool_intercepted` |
| Cognition 注入 | Kernel 与 Ranker/Evaluator/PlanPolicy 分离（`internal/cognition`） |
| 离线 parity | `geegoo verify agent-loop --offline` — 12 项，无需 config |
| 压缩血缘 | `LineageChain` + `/session` 展示（B3） |

---

## 3. Prompt、上下文与压缩

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| 四阶段压缩 | ✅ | ✅ |
| Context Engine 插件 | ✅ 可替换 | ❌ 固定 compressor |
| 辅助 LLM | ✅ `auxiliary_client` | ❌ |
| Prompt cache | Anthropic 为主 | ✅ `ApplyCacheBreakpoints` |
| 会话血缘 | SQLite FTS5 完整 | ✅ metadata + lineage chain（`/session`） |

---

## 4. Provider 与 Gateway

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| Provider 数量 | 18+ | 3（DeepSeek / OpenAI / Minimax） |
| API mode | chat / codex / anthropic | chat_completions |
| Fallback | ✅ | ✅ `SetFallbacks` |

Loop 不依赖 provider 数量；Hermes 适合多模型混用，GeeGoo 适合固定生产端点。

---

## 5. 工具执行与子 Agent

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| 注册表规模 | 70+ | ~82 |
| clarify | ✅ | ✅ + HTTP `/v1/chat/clarify` |
| 审批 / plan | ✅ approval | ✅ `approval` + `plan_gate` |
| delegate_task | ✅ | ✅ |
| **delegate_tasks 并行** | ⚠️ | ✅ `delegate_max_parallel`（默认 3） |
| 嵌套 delegate | 禁止 | ✅ `DelegateDepth >= 1` 拒绝 |
| 终端 / 浏览器 | ✅ | ❌ 刻意不做 |
| Hooks | ✅ | ✅ `config.hooks` tool_before/after |
| Schema 校验 | jsonschema 完整 | ⚠️ 子集 + 必填/嵌套校验 |

---

## 6. 可观测性与 Headless

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| TUI 进度事件 | spinner / gateway | `turn_start`、`tool_done`、`plan_proposed` 等 |
| NDJSON / CI | ⚠️ 脚本化 | ✅ `geegoo chat --message --output-format ndjson`（`schema_version: 1`） |
| EventBus | — | `TurnStarted` / `TurnCompleted` / `TurnFailed` |
| 配置发现 | — | `geegoo inspect` + `geegoo doctor` |
| 离线验收 | — | ✅ `geegoo verify agent-loop --offline` |

GeeGoo 事件表见 [agent-loop.md §3.5](../../architecture/layers/L4-runtime/agent-loop.md#35-事件与-ndjson)。

---

## 7. 设计原则对照（Hermes 六原则）

| 原则 | Hermes | GeeGooAgent |
|------|--------|-------------|
| Prompt 稳定性 | ✅ | ✅ |
| 可观测执行 | ✅ | ✅ Progress + NDJSON + EventBus |
| 可中断 | ✅ | ✅ ctx |
| 平台无关核心 | ✅ AIAgent | ✅ `Agent.Run` |
| 松耦合 | ✅ registry + plugins | ⚠️ Deps 部分硬编；Cognition 已注入 |
| Profile 隔离 | ✅ HERMES_HOME | ✅ `profiles` + `GEEGOO_PROFILE` |

---

## 8. 优劣总结

### 8.1 GeeGoo 相对 Hermes（Loop）

**优势**：双轨 Workflow + ReAct；Evidence/报告合成；`verify agent-loop` 离线 12 项；plan_gate + 并行 delegate；Go 单二进制与清晰 ctx 取消。

**劣势**：Provider 面窄；无 Context Engine / 轨迹导出；schema 校验非完整 jsonschema；无 IM/ACP；Headless 无 `-p` 短标志（有 `--message` + ndjson）。

### 8.2 Hermes 相对 GeeGoo（Loop）

**优势**：统一 ReAct 覆盖 CLI/IM/Cron/ACP；工具面广；插件生态；多 Provider。

**劣势（金融场景）**：无 Workflow Supervisor；cron 依赖 LLM 自主编排；无业务 `geegoo verify` 矩阵。

---

## 9. 选型建议

| 场景 | 建议 |
|------|------|
| 通用 IM / 编码 / 多 Provider | Hermes |
| 盘前 cron + resume + 验收 | GeeGoo Workflow + scheduler |
| 交互问股、Bot、MCP | GeeGoo `geegoo chat` |
| 从 Hermes cron 迁移 | Loop 已对齐；重点切 **workflow 路径** |

---

## 10. 参考

- GeeGoo loop：[agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md)
- 验收：[agent-loop-verification.md](../../architecture/layers/L4-runtime/agent-loop-verification.md)
- Hermes 交付：[hermes-parity-comparison.md](../../../deploy/hermes-parity-comparison.md)
- 路线图：[optimization-roadmap.md](./optimization-roadmap.md)
