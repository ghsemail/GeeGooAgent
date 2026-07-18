# L4 — Agent 循环

> Hermes 对应：[Agent 循环内部机制](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture) · `run_agent.py` / `AIAgent`

GeeGooAgent 的对话编排引擎：**Observe → Plan → Act → Update**，直到无 `tool_calls` 或达到 `max_steps`。

## 代码位置

| 组件 | 包 / 文件 |
|------|-----------|
| 对外入口 | `internal/agent/agent.go` — `Agent.Run` |
| ReAct 循环 | `internal/agent/loop.go` — `Loop.RunTurn` |
| Tool 执行 | `internal/agent/tool_exec.go` — `ToolExec`（Loop + Workflow 共用） |
| 报告合成 | `internal/agent/synthesis.go` — `ReportSynthesizer` |
| 底层派发 | `internal/runtime/executor.go` |
| 消息 sanitize | `internal/llm/messages.go` — 每轮 LLM 前合并连续 user/assistant |
| 会话消息 | `internal/runtime/session.go` |
| 稳定 System | `internal/chatprompt/builder.go` — Soul + ToolRouting + Memory + Endpoints |
| 进度回调 | `internal/runtime/progress.go` |

`Agent` 是薄封装：统一 CLI chat、HTTP runtime、未来子 Agent 的调用面。

## 单次 Turn 流程

```text
RunTurn(ctx, session, userText, toolCtx, schemas)
  │
  ├─ [可选] Compressor.ShouldCompress → 四阶段压缩
  │
  └─ loop (max_tool_rounds):
        Gateway.Chat(ctx, messages, schemas)
        ├─ 无 tool_calls → 返回最终文本
        └─ 有 tool_calls:
              ToolExec.ExecuteBatch → append tool results → 继续
        达上限 → finishBudgetExhausted（无 tool 终局总结）
```

### 与 Workflow 的分工

| 模式 | 编排者 | 适用 |
|------|--------|------|
| **ReAct** | LLM 选 Tool | `geegoo chat`、HTTP runtime |
| **确定性 Workflow** | `workflow.Runner` 硬编码步骤 | `geegoo run pre_market` |

盘前用 Workflow 是为幂等 resume、逐步验收；chat 用 ReAct 是为自然语言灵活编排。

## 核心接口（Go）

```go
// internal/agent/agent.go
func (a *Agent) Run(
    ctx context.Context,
    session *runtime.Session,
    userText string,
    toolCtx tools.Context,
    schemas []llm.ToolSchema,
) runtime.TurnResult
```

`tools.Context` 携带：

| 字段 | 用途 |
|------|------|
| `Ctx` | 取消传播（Ctrl+C） |
| `MCPToken` | GeeGoo API 鉴权 |
| `DryRun` | 跳过写操作 |
| `Interactive` + `Approved` | ApprovalGate |
| `SessionID` / `StateStore` | recall、working |

## 配置

`config.json` → `agent` 段：

| 字段 | 默认 | 说明 |
|------|------|------|
| `max_steps` | 80 | 单 turn 最大 LLM↔tool 轮数 |
| `tool_max_parallel` | 4 | 单轮并行 tool 上限（最大 16） |
| `tool_timeout_sec` | 120 | 单次 tool 超时秒数（最大 600） |
| `temperature` | 0.2 | 传给 Provider |
| `context_token_budget` | 模型相关 | 压缩阈值参考 |

压缩阈值见 `internal/prompt/compressor.go`（回合开始 85%、每轮 LLM 前 50%）。

## 可观测性

- `SetProgress(fn)` — chatui spinner / 工具预览
- Hermes 对齐回调：`thinking_start/stop`、`step_complete`、`tool_gen_start/delta`、`stream_delta`
- `Result.Meta` — HTTP 工具 `api_code`、`duration_ms`
- EventBus（L0）— workflow 路径发 `ToolCalled` / `ToolCompleted`；chat 路径发 `TurnStarted`/`TurnCompleted`；报告合成发 `SynthesisStarted`/`SynthesisCompleted`

## 可中断性

`context.Context` 贯穿 `Gateway.Chat` 与 `Registry.Execute`。用户 Ctrl+C 中断**当前回合**；会话历史已持久化，下回合可继续。

## 测试

- `internal/agent/agent_test.go`
- `internal/agent/loop_test.go`
- `internal/agent/loop_interrupt_test.go`
- `internal/agent/loop_compress_test.go`

## 延伸阅读

- [workflow-engine.md](./workflow-engine.md) — 确定性工作流
- [../../entrypoints.md](../../entrypoints.md) — CLI / HTTP 如何调用 Agent
- [../L1-model-gateway/gateway.md](../L1-model-gateway/gateway.md) — LLM 调用链

## 设计意图（Observe → Plan → Act → Update）

循环直到无 `tool_calls` 或 `max_steps`：

1. **Observe** — `RuntimeMessages` + Working 状态组装上下文；必要时压缩（`internal/prompt`）
2. **Plan** — `gateway.Chat` + tool schemas
3. **Act** — `Executor` 调 `tools.Registry`；写 session tool 消息
4. **Update** — Working/Evidence；workflow 路径另写 checkpoint

配置项见 `config.json` 的 `agent.max_tool_rounds`、`prompt` 压缩阈值。
