# Agent Runtime 架构

> **状态**：2026-07 定稿并已落地  
> **定位**：GeeGooAgent 作为 **Agent Runtime / Agent OS** 的技术架构 SSOT。  
> **实现对照**：[implementation-status.md](./implementation-status.md) 与 `internal/`。  
> **未做项**：[backlog.md](./backlog.md)（唯一待办清单）。

与 [六层模型](./overview.md) 的关系：六层描述**能力分层**；本文描述**控制面 / 语言边界 / 包边界**。冲突时以本文硬边界为准。

---

## 1. 一句话结论

**GeeGooAgent 是 Agent Runtime（Go 控制面），不是 chatbot 框架。**  
Python 仅作可选 **Advisor**（默认关闭）；客户端为 CLI/TUI 与 `runtimeapi`。

核心竞争力是 **Runtime 可靠性**：权限、超时、取消、流式、持久化、审计、预算、可恢复会话。

---

## 2. 架构图

```text
                         Clients
                    CLI / TUI  │  runtimeapi（GeeGooBot 等）
                         │
                   runtimeapi :3400
                         │
         ┌───────────────▼───────────────┐
         │        Agent Kernel (Go)       │
         │   loop / state / event / gate  │
         │   internal/agent               │
         └───────────────┬───────────────┘
                         │
     ┌───────────────────┼───────────────────┐
     ▼                   ▼                   ▼
 workflow            cognition            runtime
 internal/workflow   internal/cognition   internal/runtime
 (确定性编排)         Ranker/Evaluator/     Executor/Session
                     PlanPolicy
                         │
              Go default ──optional──► Python Advisor
              (IdentityRanker等)        services/cognitive
                                        (suggestion only)


 runtime 子系统（全部 Go）:

   tools + sandbox + MCP          internal/tools, clients/mcp
   session store (SSOT)           internal/chatsession + SQLite
   memory port                    internal/memport + memory.Adapter
   model policy + gateway         internal/llm (policy.go, gateway.go)
```

---

## 3. 硬边界（不可破）

| # | 规则 |
|---|------|
| 1 | **Kernel 拥有 Loop**；Python 永不拥有 state / tool / workflow 决策 |
| 2 | **cognition ≠ kernel**；策略经 `SetCognition` 注入，不进 loop 实现细节 |
| 3 | **Python = Advisor**：只读 snapshot → suggestion；禁止 tool call、写 session、workflow 决策 |
| 4 | **Session SSOT ≠ Memory 全部落库**：SQLite 管 conversation / task / trace；向量库仅为可重建索引 |
| 5 | **出站模型经 Policy → Gateway**；策略与 provider 适配分离 |
| 6 | **包边界机器可验**：`go run scripts/check_import_boundaries.go`（见 [engineering/agent-runtime-boundaries.md](../engineering/agent-runtime-boundaries.md)） |

### 明确禁止

- ReAct loop 迁到 Python  
- Python 直连 provider 绕过 Gateway  
- Python / UI 直接执行 tool / 写 session  
- 「long reasoning」第二套可突变状态的 Agent  
- 为 Dashboard 提前改 Kernel 契约  

---

## 4. 语言与 Ownership

| 层 | 语言 | Ownership |
|----|------|-----------|
| Agent Kernel | Go | **唯一** |
| Tool Runtime（sandbox / MCP） | Go | **唯一** |
| Model Policy + LLM Gateway | Go | **唯一** |
| Workflow | Go | **唯一** |
| Session Store | Go | **唯一 SSOT** |
| Memory port / Adapter | Go | 写路径由 Go 决定；后端可演进 |
| Cognition 默认实现 | Go | `internal/cognition` |
| Cognition Advisor | Python（可选） | **无**；失败降级 |
| CLI / TUI | Go | 客户端 |
| Dashboard | （待办） | 纯客户端 → [backlog.md](./backlog.md) |

降级原则：**Kernel 健康 ≠ Plugin 健康**。

---

## 5. 逻辑包与代码落点

| 逻辑包 | 职责 | Go 包 |
|--------|------|-------|
| Kernel | loop、tool exec、approval、预算 | `internal/agent`、`internal/runtime` |
| Cognition | Ranker / Evaluator / PlanPolicy；可选 Advisor | `internal/cognition` |
| Tool runtime | Registry、MCP、sandbox | `internal/tools`、`internal/clients/mcp` |
| Model | Policy + Gateway + providers | `internal/llm` |
| Session | 对话 / plan / 轨迹 SSOT | `internal/chatsession` |
| Memory port | Recall / Store / Compress | `internal/memport`、`internal/memory` |
| Workflow | 确定性业务流 | `internal/workflow` |
| API / CLI | HTTP、TUI、REPL | `internal/runtimeapi`、`internal/cli` |
| 边界检查 | import 规则 | `internal/archboundaries` |

### 依赖方向

```text
api / cli → app → agent
agent → cognition | workflow | tools | llm | memport | prompt | runtime

cognition  ↛ agent | cli | api | tools | app
tools      ↛ cognition
memport    ↛ memory | tools | agent
```

Recall 排序链：`memory.Adapter.SessionRanker` → `agent.RankRecallHits` → `cognition.Ranker`（`tools` 不 import `cognition`）。

---

## 6. Kernel 与 Cognition

```text
Loop（控制平面）
  ├── ContextBuilder     → chatprompt + compressor（Memory port）
  ├── PlanPolicy         → plan gate hold / 文案
  ├── Executor           → ToolExec + Approval
  ├── Memory port        → Compress / Recall / Store(evidence)
  └── Cognition hooks    → Ranker / Evaluator（回合末评估）
```

- **Loop ≠ Planner**：ReAct 中 LLM `tool_calls` 仍是默认计划来源；`PlanPolicy` 只管门控策略。  
- **Ranker**：`recall` tool 经 Memory port 检索后由 Ranker 重排；Advisor 开启时可走 Python rank。  
- **Evaluator**：回合结束 advisory 判断；默认 `AcceptAllEvaluator`。

### Python Advisor 契约

| 允许 | 禁止 |
|------|------|
| `POST /v1/advisor/rank`、`/evaluate` | `tool_calls`、state 写入、workflow 决策 |
| 排序、评分、重试建议 | 成为第二 operator |

配置：`config.advisor`（默认 `enabled: false`）。Sidecar：`services/cognitive/advisor_server.py`；可选 `deploy/systemd/geegoo-advisor.service`。

---

## 7. Session 与 Memory

| 概念 | 实现 |
|------|------|
| **Session SSOT** | `chatsession` + SQLite：消息、pending plan、tool 轨迹 |
| **Memory port** | `memport.Port`：`Recall` / `Store` / `Compress` |
| **Adapter** | `memory.Adapter` 委托 Compressor、SessionStore、EvidenceStore |
| **向量 / 语义索引** | 未做 → [backlog.md](./backlog.md) |

---

## 8. Model Policy 与 Gateway

```text
Kernel / report / compressor
       │  context.CallMeta (TaskKind)
       ▼
 ComplexityPolicy → ConfigPolicy    ← App 默认栈
       │
       ▼
 Gateway（重试、流式、fallback）
       │
       ▼
 Providers
```

| TaskKind | 典型用途 |
|----------|----------|
| `chat` | ReAct 主循环 |
| `compress` | 上下文压缩摘要 |
| `synthesis` | 报告合成 |
| `complex` | budget summary 等；可抬 max_tokens |

`ComplexityPolicy` 默认 `ToolSchemaThreshold=0`，避免 82 tools 误抬全员 `max_tokens`；仅 `TaskComplex` 触发抬升。

---

## 9. 与六层模型对照

| 六层 | 与本文 |
|------|--------|
| L4 Runtime | Kernel + Workflow + Cognition 调用 |
| L3 Memory | session + memport + memory.Adapter |
| L2 Tools | toolruntime |
| L1 Gateway | llm policy + gateway |
| L0 Infra | SQLite、EventBus、Scheduler、Sandbox |
| L5 Application | Skill、Rules、CLI |

层文档：[layers/](./layers/) 各 README。

---

## 10. 客户端优先级

| 优先级 | 客户端 | 状态 |
|--------|--------|------|
| 1 | CLI / TUI | ✅ 维护 |
| 2 | `runtimeapi` | ✅ 维护 |
| 3 | IDE 扩展 | 待办 |
| 4 | Dashboard / Flutter | 待办 → [backlog.md](./backlog.md) |

---

## 11. 仓库布局

```text
GeeGooAgent/
├── cmd/geegoo/、cmd/agent-runtime/
├── internal/
│   ├── agent/           # Kernel
│   ├── cognition/       # 策略 + AdvisorClient
│   ├── memport/         # Memory 接口
│   ├── memory/          # Adapter、Working、Evidence
│   ├── llm/             # Policy + Gateway
│   ├── tools/、clients/mcp/
│   ├── chatsession/、prompt/、chatprompt/
│   ├── runtime/、workflow/、runtimeapi/、cli/
│   ├── archboundaries/
│   └── app/             # 组装：wireChatMemory、wireCognition、wireRecallRanker
├── services/cognitive/  # 可选 Python Advisor
├── scripts/check_import_boundaries.go
├── skills/、rules/、deploy/
└── .github/workflows/ci.yml
```

详见 [repo-layout.md](./repo-layout.md)。

---

## 12. 验收

```bash
go run scripts/check_import_boundaries.go
go test ./internal/cognition/... ./internal/agent/... ./internal/llm/...
         ./internal/memport/... ./internal/memory/... ./internal/archboundaries/...
go build ./cmd/geegoo ./cmd/agent-runtime
geegoo verify agent-loop    # 运行时验收见 agent-loop-verification.md
```

---

## 相关文档

- [overview.md](./overview.md) — 六层导图与数据流  
- [repo-layout.md](./repo-layout.md) — 目录与包对照  
- [implementation-status.md](./implementation-status.md) — 实现状态 SSOT  
- [backlog.md](./backlog.md) — 唯一待办  
- [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) — 循环实现  
- [engineering/agent-runtime-boundaries.md](../engineering/agent-runtime-boundaries.md) — 工程边界与禁止清单  
