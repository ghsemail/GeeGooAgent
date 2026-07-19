# Agent Loop：GeeGooAgent × Grok Build

> 更新：2026-07-20。GeeGoo 以 [agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md) 为准；Grok 以 [开源仓库](https://github.com/xai-org/grok-build)、[docs.x.ai/build](https://docs.x.ai/build/overview) 及 [grok-build 功能整理](../grok-build.md) 为准。

## 文档概述

本文档对比 **编码 harness**（Grok Build / `xai-grok-shell`）与 **金融 Agent Loop**（GeeGoo `internal/agent`）在 ReAct 机制、Plan 门控、子 Agent、Headless 与工具执行环境上的差异。用于判断哪些 Grok 能力值得借鉴、哪些与 GeeGoo 定位冲突。三方速查见 [comparison.md](../comparison.md)；GeeGoo 落地计划见 [optimization-roadmap.md](./optimization-roadmap.md)。

## 摘要

| 维度 | 结论 |
|------|------|
| **产品形态** | Grok = 编码 harness；GeeGoo = 金融 MCP 编排 + 可选 Workflow |
| **Loop 骨架** | 同为 ReAct；Grok 强化 **编码 Plan mode、worktree 子 Agent、文件/终端工具** |
| **GeeGoo 已对齐** | plan_gate（mutating）、`delegate_tasks` 并行、Hooks、NDJSON Headless、压缩/缓存/clarify、offline verify |
| **Grok 仍更强** | 结构化编码 Plan（`.grok/plan.md`）、worktree 隔离、`grok -p`、文件/Git/沙箱工具、ACP |

---

## 1. 架构对照

```text
Grok Build (Rust)                   GeeGooAgent (Go)
xai-grok-shell      → Agent runtime internal/agent/loop.go
xai-grok-tools      → 工具           internal/tools/ + MCP
xai-grok-workspace  → 检查点         workflow checkpoint + chatsession
```

GeeGoo 用 `App` 聚合 Gateway、Registry、Workflow；Grok 拆 harness / tools / workspace crate。

### 运行形态

| 形态 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 交互 TUI | ✅ `grok` | ✅ `geegoo chat` |
| Headless 对话 | ✅ `grok -p` + streaming-json | ⚠️ `geegoo chat --message --output-format ndjson`（无 `-p` 别名） |
| 确定性批处理 | ⚠️ 仍走 Agent | ✅ `workflow.Runner` + `geegoo verify` |
| ACP / IDE | ✅ | ❌ |

---

## 2. Agent Loop 机制

### 2.1 ReAct 环（共同）

```text
messages + tools → model(stream) → tool_calls? → execute → append → repeat
```

| 环节 | Grok (`xai-grok-shell`) | GeeGoo (`Loop`) |
|------|-------------------------|-----------------|
| 并行 tool | ✅ | ✅ `tool_max_parallel` |
| 步数上限 | ✅ | ✅ `max_steps`（硬顶 90） |
| 预算耗尽 | ✅ | ✅ `finishBudgetExhausted` |
| 压缩 | ✅（未公开细节） | ✅ Hermes 四阶段 + 双阈值 |
| Prompt cache | 端点依赖 | ✅ `ApplyCacheBreakpoints` |

### 2.2 Plan：编码 Plan mode vs 金融 plan_gate

| | Grok Build | GeeGooAgent |
|---|------------|-------------|
| **目标** | 改代码前先审计划 + diff | mutating API（create/update/delete bot 等）先确认 |
| **机制** | Plan mode 状态机；默认 `.grok/plan.md`；批准前禁止编辑 | `plan_gate` + `session.PendingPlan` + `plan_proposed` |
| **范围** | 全仓库编码任务 | 金融写操作 tool |
| **对比** | harness 层强制 | Loop 内挂起 tool_calls，用户 `y`/`n` 或 HTTP `/v1/chat/plan` |

**结论**：GeeGoo **已实现**写操作门控，但 **不是** Grok 式「先写 plan.md 再改文件」；编码 Plan mode 仍不适用。

### 2.3 子 Agent 与并行

| 能力 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 委派 | `task` / spawn | `delegate_task` / `delegate_tasks` |
| **并行子 Agent** | ✅ ~8 路 + worktree | ✅ `delegate_tasks` + `delegate_max_parallel`（默认 3，最大 8） |
| 独立预算 | ✅ | ✅ `sub_agent_max_steps` |
| worktree 隔离 | ✅ | ❌ |
| 子 Agent 指定模型 | ✅ | ❌ |

盘前多标的并行：GeeGoo 可用 **Workflow 代码编排** 或 **delegate_tasks**；Grok 偏编码 worktree。

### 2.4 Headless 与验收

| 能力 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 单命令对话 | ✅ `grok -p "..."` | ⚠️ `geegoo chat --message "..." --output-format ndjson` |
| 结构化流 | ✅ `streaming-json` | ✅ NDJSON `schema_version: 1` |
| Loop 离线验收 | — | ✅ `geegoo verify agent-loop --offline`（12 项） |
| 业务字段验收 | — | ✅ `geegoo verify` 矩阵 |

CI 嵌入「一句话修 bug」→ Grok；验收盘前/loop 契约 → GeeGoo。

### 2.5 Evaluator 质量闭环

| | Grok | GeeGoo |
|---|------|--------|
| 回合后质检重试 | —（未公开） | ⚠️ `eval_max_retries`（0–1）+ 可选 Advisor `RetrySuggested` |

---

## 3. 工具执行环境（Act 层）

| 工具类 | Grok | GeeGoo |
|--------|------|--------|
| 文件 patch / Git / Shell | ✅ | ❌ 刻意不做 |
| 沙箱执行 | ✅ | ❌ |
| 金融 MCP ~82 | — | ✅ |
| Hooks | ✅ | ✅ `config.hooks` |
| 写操作审批 | ✅ | ✅ approval + plan_gate |

Grok loop 围绕**本地工作区**；GeeGoo 围绕 **远程 MCP/API**。

---

## 4. 上下文与检查点

| 能力 | Grok | GeeGoo |
|------|------|--------|
| 会话持久化 | ✅ | ✅ SQLite `chatsession` |
| 编码回滚检查点 | ✅ workspace | — |
| Workflow resume | — | ✅ checkpoint + `CompletedStepKeys` |
| Evidence 审计 | — | ✅ `evidence_records` |

---

## 5. 优劣总结

### 5.1 GeeGoo 相对 Grok（Loop）

**优势**：金融工具域；Workflow + Supervisor；plan_gate + 并行 delegate；NDJSON + offline verify；Evidence；单 Go 二进制。

**劣势**：无编码 Plan mode / worktree；无文件/终端/Git；无 `grok -p`；无 ACP。

### 5.2 Grok 相对 GeeGoo（Loop）

**优势**：编码 harness 完整度；Headless `-p`；ACP；AGENTS.md 生态。

**劣势**：无盘前 Workflow/verify；无金融 MCP；scheduler+verdict 非一等公民。

---

## 6. 可借鉴项（2026-07 状态）

| 优先级 | Grok 能力 | GeeGoo 现状 | 备注 |
|--------|-----------|-------------|------|
| — | Plan mode（编码） | — | 不做 |
| — | 文件/终端/Git | — | 不做 |
| P1 | 写操作门控 | ✅ `plan_gate` | 已落地 |
| P2 | Headless JSON | ⚠️ NDJSON 已有；缺 `-p` 别名 | 见 roadmap A1 |
| P3 | `inspect` | ✅ `geegoo inspect` | 已落地 |
| P4 | 并行 delegate | ✅ `delegate_tasks` | 已落地 |
| P5 | Hooks | ✅ `config.hooks` | 已落地 |
| P6 | Cost / token 预算 | ❌ | roadmap Phase D |
| P7 | Evaluator 重试 | ⚠️ `eval_max_retries` | 2026-07 最小闭环 |

详情：[optimization-roadmap.md](./optimization-roadmap.md)

---

## 7. 选型建议

| 场景 | 建议 |
|------|------|
| 终端改代码、跑测试、开 PR | Grok Build |
| 盘前、信号、Bot、MCP | GeeGooAgent |
| CI 一句话修 bug | Grok `grok -p` |
| CI 验证 loop / 盘前字段 | GeeGoo `verify` |

---

## 8. 参考

- Grok 功能清单：[../grok-build.md](../grok-build.md)
- 三方对比：[../comparison.md](../comparison.md)
- GeeGoo loop：[../../architecture/layers/L4-runtime/agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md)
- 验收：[../../architecture/layers/L4-runtime/agent-loop-verification.md](../../architecture/layers/L4-runtime/agent-loop-verification.md)
