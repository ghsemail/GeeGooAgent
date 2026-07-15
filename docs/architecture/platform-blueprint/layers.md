# 六层接口规范

> 智能体实现每层时**必须提供下列接口**（Go 签名见 GeeGooAgent；此处为语言无关 SSOT）。

---

## L0 — Infrastructure

### 职责

进程级能力：持久化、事件、检查点、调度适配、沙箱策略、日志。  
**不提供** LLM、Tool、业务解析。

### 必做模块（Phase 0）

| 模块 | 接口 | MVP 实现 |
|------|------|----------|
| EventBus | `Emit(event, payload)` | 内存同步 |
| StateStore | `Save(key, map)` / `Load(key)` | JSON 文件 |
| CheckpointManager | `Save(Checkpoint)` / `Load(sessionID)` | 同 StateStore |
| Scheduler | 文档 + systemd unit | 不调内置 cron |

### Checkpoint 结构

```json
{
  "session_id": "uuid",
  "skill": "first_skill",
  "step": 12,
  "status": "running",
  "last_tool": "fetch_data",
  "working": { }
}
```

### 依赖规则

- 被 L2/L3/L4 使用
- **不得** import runtime / tools / llm

---

## L1 — Model Gateway

### 职责

统一 LLM 入口：重试、超时、温度、max_tokens、多 Provider 切换。

### 接口

```text
Provider.chat(messages, tool_schemas, temperature, max_tokens) → Response
Gateway.chat(messages, tool_schemas, session_id, step) → Response  // 内含重试
```

```text
Message: { role, content, tool_calls?, tool_call_id? }
Response: { content, reasoning_content?, tool_calls[] }
ToolSchema: { name, description, parameters JSON Schema }
```

### MVP

- 1 个 OpenAI 兼容 Provider
- `MaxRetries: 3`, `RetryWait: 5s`
- `mock.Provider` 供测试

### 禁止

- Runtime 直接 `openai.ChatCompletion`
- Gateway 感知业务 Tool 名

---

## L2 — Tools

### 职责

注册、过滤、执行 Tool；导出 JSON Schema 给 LLM；封装外部 HTTP。

### 核心类型

```text
ToolContext:
  session_id, user_token, dry_run, step
  event_bus, state_store, workspace_root

ToolResult:
  status: ok | error | dry_run | skipped
  summary: string          // ≤300 字，给 LLM
  data: map                // 给 Working.Apply
  exit_code: int

Tool:
  name, description, parameters, handler(ctx, args) → ToolResult

ToolRegistry:
  register(tool)
  schemas(filter_names?) → []ToolSchema
  execute(call, ctx) → ToolResult
```

### Catalog 模式（推荐）

声明式 HTTP Tool，避免手写 50 个 wrapper：

```yaml
# catalog 条目示例
- name: check_condition
  path: /checkCondition
  method: POST
  needs_user_token: true
  merge_payload: [code]
```

`RegisterHTTPFromCatalog(registry, client, catalog)` 批量注册。

### Bespoke 模式

本地文件、复合逻辑、调用多个 API → `bespoke.go` 手写 Handler。

### Tool 过滤（按 Skill 模式）

| mode | 排除 |
|------|------|
| `scheduled` | `create_*`, `delete_*` |
| `interactive` | 无（危险操作 Tool 内二次确认） |
| `signal` | CRUD 类 |

### MVP Tool 数量

- Phase 0：**1** mock + **1** HTTP echo
- Phase 1：**首个 Skill 所需全部**（通常 10–20）
- 目标态：Catalog 自动生成 + 少量 Bespoke

---

## L3 — Memory

### 职责

| 类型 | 用途 | MVP |
|------|------|-----|
| **Session** | ReAct 对话 messages | Phase 0 内存；Phase 2 持久化 |
| **Working** | Workflow 结构化状态 | Phase 1 必须 |
| Episodic | 跨日 md/jsonl | stub |
| Semantic | 向量检索 | stub |

### Working 模式（关键）

```text
WorkingStore:
  create(session_id, skill) → Working
  load(session_id) → Working
  save(working)
  apply(working, tool_name, result) → Working   // 核心
```

**Apply 规则**：每个 `tool_name` 一个分支，从 `result.Data` 提取字段更新 Working。  
**禁止**：把完整 API JSON  append 到 messages。

GeeGooAgent 参考：`internal/memory/working.go` 中 `Apply()` switch。

---

## L4 — Agent Runtime

### 4.1 Executor

```text
Executor.execute(call, ctx) → ToolResult
  → registry.execute(call, ctx)
```

Workflow 与 ReAct **共用** Executor。

### 4.2 ReActLoop

```text
ReActLoop.run_turn(session, user_text, ctx, schemas) → TurnResult:
  append user message
  for round in 1..max_tool_rounds:
    response = gateway.chat(messages, schemas)
    if no tool_calls:
      return assistant text
    for each call:
      result = executor.execute(call, ctx)
      append tool message (summary + truncated data)
    optional: checkpoint.save
  return error max_rounds
```

配置：`max_tool_rounds`（chat 建议 8–20）、`tool_result_max_chars`（建议 6000）。

### 4.3 WorkflowRunner

```text
Step:
  name, tool, arguments | arg_func(working → arguments)

WorkflowRunner.run(session_id, skill, phase_a[], per_item[], ctx, working):
  for step in flatten(phase_a, per_item_per_entity):
    result = executor.execute(step.tool, step.args(working), ctx)
    working = working_store.apply(working, step.tool, result)
    checkpoint.save(session_id, step_index, working)
    if short_circuit(working): return completed
    if result.status == error: return failed
  return completed
```

**Phase A**：全局步骤（检查条件、拉列表、全局数据）。  
**Phase B**：对每个 entity 重复 `per_item` 步骤。

### 4.4 Session

```text
Session:
  id, skill, step_counter, messages[], status
  append_message(), llm_messages()
```

---

## L5 — Application

### 职责

- Skill Pack（manifest + 文档）
- CLI / HTTP 入口
- Rules / Prompts 组装 system prompt

### System Prompt 组装顺序

```text
prompts/identity.md
+ rules/*.md（有序）
+ skills/<name>/SKILL.md
+ skills/<name>/workflow.md（摘要）
```

### SkillLoader（Phase 3）

```text
LoadedSkill:
  name, mode, tool_filter[], workflow_steps[], supervisor_checks[]

SkillLoader.load(name, mode) → LoadedSkill
  → 读 manifest.yaml + 解析 supervisor_checks.yaml
```

Phase 1 允许硬编码 `workflow/steps.go`；Phase 3 必须改为 loader。

---

## 横切 — Supervisor

```yaml
# supervisor_checks.yaml 示例
checks:
  - id: trading_day_gate
    when: "working.is_trading_day == false"
    assert: "status == completed"
  - id: all_stocks_reported
    when: "phase == done"
    assert: "all(stocks, status in [reported, skipped])"
```

```text
Supervisor.verify(working, checks) → []Violation
```

Phase 1 可人工抽检；Phase 3 自动化。

---

## 层间调用合法表

| 调用方 | 允许调用 |
|--------|----------|
| cmd | app, config |
| app | 所有 internal 包（wiring） |
| runtime/workflow | llm/gateway, tools/executor, memory, infra |
| tools | clients, infra |
| llm | 无 internal 上层 |
| infra | 无 internal 上层 |
