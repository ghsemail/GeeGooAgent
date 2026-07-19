# Agent Runtime 架构定稿

> **状态**：2026-07-19 定稿  
> **定位**：GeeGooAgent 作为长期 **Agent Runtime / Agent OS** 的技术架构 SSOT。  
> **实现对照**仍以 [implementation-status.md](./implementation-status.md) 与 `internal/` 为准。  
> **改造节奏**见 [agent-runtime-migration-plan.md](./agent-runtime-migration-plan.md)。

与既有 [六层模型](./overview.md) 的关系：六层描述**能力分层**；本文描述**控制面 / 语言边界 / 包边界**。二者并存，冲突时以本文的硬边界为准。

---

## 1. 一句话结论

**GeeGooAgent 是 Agent Runtime（Go 控制面），不是 chatbot 框架。**  
Python 只做可选 **Advisor**；客户端优先 CLI/TUI 与现有 `runtimeapi`；Dashboard（含 Flutter）**本期不做，后续单独规划**。

核心竞争力不是「会不会调 LLM」，而是 **Runtime 可靠性**：权限、超时、取消、流式、持久化、审计、预算、可恢复会话。

---

## 2. 定稿架构图

```text
                         Clients（本期）
                    CLI / TUI  │  runtimeapi 调用方（如 GeeGooBot）
                         │
                   runtimeapi :3400
                         │
         ┌───────────────▼───────────────┐
         │        Agent Kernel (Go)       │
         │   loop / state / event / gate  │
         │   不内嵌具体认知策略实现          │
         └───────────────┬───────────────┘
                         │
     ┌───────────────────┼───────────────────┐
     ▼                   ▼                   ▼
 workflow            cognition            runtime
 (确定性编排)         (策略扩展点)          (执行与真相源)
                         │
              ┌──────────┼──────────┐
              ▼          ▼          ▼
           planner    ranker    evaluator
              │
         Go default ──optional──► Python Advisor
                                   (suggestion only)


 runtime 子系统（全部 Go，可信边界内）:

   tools + sandbox + MCP
   session store          ← 对话/任务/轨迹 SSOT（当前 = SQLite）
   memory ports           ← Recall/Store/Compress；实现可多后端
   model runtime/policy   ← 选模、预算、并行、是否压缩
        └── llm gateway   ← provider 适配、流式归一、fallback
```

**后续（不在本期改造范围）**：Web / Flutter Dashboard、IDE 扩展 — 一律只消费 `runtimeapi`，不内嵌 agent 逻辑。见迁移计划「后续规划」。

---

## 3. 硬边界（不可破）

| # | 规则 |
|---|------|
| 1 | **Kernel 拥有 Loop**；Python 永不拥有 state / tool / workflow 决策 |
| 2 | **cognition ≠ kernel**；Planner / Ranker / Evaluator 是策略，不进 loop 实现包 |
| 3 | **Python = Advisor**：`in: context snapshot` → `out: suggestion`；禁止 tool call、禁止写 session、禁止 workflow 决策 |
| 4 | **Session SSOT ≠ Memory 全部落库**：SQLite 管 conversation / task / trace；向量库 / 对象存储只做可重建索引与制品 |
| 5 | **出站模型只经 Model Runtime/Policy → LLM Gateway**；策略层与 provider 适配层分离 |
| 6 | **用逻辑包边界防 Go 泥球**：禁止跨层乱 import；物理目录可渐进拆分 |

### 明确禁止

- 把 ReAct loop 迁到 Python  
- Python 直连 provider 绕过 Gateway（默认）  
- Python 或 UI 直接执行 tool / 写 session  
- 把「long reasoning」做成可突变状态的第二套 Agent  
- 为 Dashboard 提前改 Kernel 契约（Dashboard 后置）

---

## 4. 语言与 Ownership

| 层 | 语言 | Ownership |
|----|------|-----------|
| Agent Kernel | Go | **唯一** |
| Tool Runtime（含 sandbox / MCP） | Go | **唯一** |
| Model Policy + LLM Gateway | Go | **唯一** |
| Workflow | Go | **唯一** |
| Session Store | Go | **唯一 SSOT** |
| Memory ports / 实现 | Go（索引后端可演进） | 写路径与生命周期由 Go 决定 |
| Cognition 默认实现 | Go | 默认路径 |
| Cognition Advisor sidecar | Python（可选） | **无**；失败必须降级 |
| CLI / TUI | Go | 客户端 |
| Dashboard | （后续） | 纯客户端，无 ownership |

降级原则：**Kernel 健康 ≠ Plugin 健康**；Plugin 挂掉不得拖垮 chat / workflow。

---

## 5. 逻辑包边界

逻辑名（防泥球）与当前/目标代码对应：

| 逻辑包 | 职责 | 当前主要落点 | 目标落点 |
|--------|------|--------------|----------|
| `agentkernel` | loop、state、event、approval、预算调度 | `internal/agent`（loop_*）、`internal/runtime` | 保持 Kernel 薄；只依赖 cognition 接口 |
| `cognition` | planner / ranker / evaluator 接口 + Go 默认 | 散落在 loop / plan_gate / prompt | `internal/cognition/`（新建） |
| `toolruntime` | tools、sandbox、MCP、audit | `internal/tools`、`clients/mcp` | 不变；禁止 cognition 反向依赖 |
| `model` | policy/runtime + gateway + providers | `internal/llm` | 先抽 Policy，再视需要分子包 |
| `session` | conversation / task / step SSOT | `internal/chatsession` | 接口化存储，当前实现 SQLite |
| `memory` | Memory port；shortterm 等 | `internal/memory`、`prompt` | 接口先行；多后端后置 |
| `workflow` | 确定性业务流 | `internal/workflow` | 不变 |
| `api` | HTTP | `internal/runtimeapi` | 不变 |
| `cli` | TUI / REPL | `internal/cli/*` | 不变 |

### 依赖方向（只允许）

```text
api / cli
  → agentkernel
      → cognition | workflow | toolruntime | model | session | memory

cognition  不得 import  cli / api
toolruntime 不得 import  cognition
Python sidecar 只被 cognition 的某一实现调用
```

---

## 6. Kernel 与 Cognition

```text
Loop（控制平面，Go 固定）
  │
  ├── ContextBuilder     → chatprompt + compressor
  ├── Planner            → 默认：LLM tool_calls / 结构化 plan
  ├── Executor           → ToolExec + Approval + sandbox
  ├── MemoryStore        → session / memory（写路径只在 Go）
  └── Cognition hooks    → Ranker / Evaluator（可换 Advisor）
```

**Loop ≠ Planner**：Loop 管生命周期；Planner 只产出「下一步意图」。  
ReAct 中 LLM 的 `tool_calls` 仍是默认 Plan 来源；可抽的是门控、分解、排序、事后评估等**策略**，不是整段循环。

### Python Advisor 契约（窄）

**允许**

```text
input:  context snapshot（只读摘要）
output: suggestion（排序、评分、可选重试建议、prompt 片段建议）
```

**禁止**

```text
tool call · state mutation · workflow decision · 成为第二套 operator
```

---

## 7. Session 与 Memory

| 概念 | 定义 |
|------|------|
| **Session SSOT** | 对话消息、pending plan、tool 轨迹、可恢复任务状态；**当前实现 = SQLite** |
| **Memory** | `Recall` / `Store` / `Compress` 端口；可演进 shortterm / episodic / semantic / procedural |
| **索引 / 制品** | Vector DB、对象存储等为可重建层，**不是** conversation 真相源 |

不要把「SQLite 是 SSOT」误解为「所有记忆都只能进 SQLite」。

---

## 8. Model Runtime / Policy 与 Gateway

```text
Kernel / Cognition
       │
       ▼
 Model Runtime / Policy     ← 选哪个模型、temperature、budget、是否压缩、并行、fallback
       │
       ▼
 LLM Gateway                ← provider 协议、流式归一、重试
       │
       ▼
 Providers (DeepSeek / OpenAI / …)
```

今日 `internal/llm.Gateway` 已部分承担 Runtime 职责；定稿要求**策略与适配分离**，避免 Gateway 文件无限膨胀。

---

## 9. 与六层模型对照

| 六层 | 与本文关系 |
|------|------------|
| L4 Runtime | ≈ Kernel + Workflow +（调用）Cognition |
| L3 Memory | ≈ session + memory ports |
| L2 Tools | ≈ toolruntime |
| L1 Gateway | ≈ model（policy + gateway） |
| L0 Infra | SQLite / EventBus / Scheduler / Sandbox 支撑 runtime |
| L5 Application | Skill / Rules / CLI；不拥有 Kernel |

既有层文档继续有效；新增能力按本文硬边界落点，避免再把策略写进 loop 文件。

---

## 10. 客户端优先级（产品）

| 优先级 | 客户端 | 本期 |
|--------|--------|------|
| 1 | CLI / TUI（`geegoo chat`） | ✅ 维护 |
| 2 | `runtimeapi`（Bot 等） | ✅ 维护 |
| 3 | IDE 扩展 | 后续 |
| 4 | Web / Flutter Dashboard | **后续单独规划，本期不做** |

---

## 11. 仓库目标布局（渐进，非一夜搬家）

```text
GeeGooAgent/
├── cmd/geegoo/
├── cmd/agent-runtime/
├── internal/
│   ├── agent/           # Kernel 表面（loop 等）；变薄
│   ├── cognition/       # ★ 策略接口 + Go 默认（改造引入）
│   ├── runtime/
│   ├── workflow/
│   ├── llm/             # Gateway；Policy 可先同包后拆
│   ├── tools/ + clients/mcp/
│   ├── chatsession/
│   ├── memory/
│   ├── chatprompt/ + prompt/
│   ├── runtimeapi/
│   └── cli/
├── services/            # ★ 可选；P2+ 再引入
│   └── cognitive/       # Python Advisor sidecar
├── skills/  rules/  deploy/ docs/
└── go.mod
```

`web/dashboard`：**不在本期创建不建目录承诺。

---

## 12. 成功标准

- 无 Python 时行为与今日等价（默认全 Go）  
- cognition 可替换实现而不改 Loop 状态机  
- Session 可恢复；Plugin 挂掉 chat 仍可用  
- import 图符合第 5 节依赖方向  
- Dashboard 未引入前，不阻塞 Kernel / cognition / model 改造  

---

## 相关文档

- [改造计划](./agent-runtime-migration-plan.md)  
- [overview.md](./overview.md) — 六层导图  
- [repo-layout.md](./repo-layout.md) — 仓库布局（随改造更新）  
- [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) — 循环实现  
- [layers/L1-model-gateway/](./layers/L1-model-gateway/) — Gateway 现状  
- [layers/L3-memory/](./layers/L3-memory/) — Memory 现状  
