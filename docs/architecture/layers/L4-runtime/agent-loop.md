# L4 — Agent 循环

> 更新：2026-07-19（已部署 `225615ed` → 119.45.16.112）。Hermes 对应：[Agent 循环内部机制](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture) · `run_agent.py` / `AIAgent`

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
| NDJSON 事件 | `internal/runtime/agent_events.go` + `internal/cli/progress/ndjson.go` |
| Loop 自检 | `geegoo inspect` → `internal/inspect/report.go` |
| HTTP clarify | `internal/runtimeapi/clarify_hub.go` + `POST /v1/chat/clarify` |

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
| `DelegateDepth` / `Progress` / `ClarifyFn` / `Hooks` | 子 Agent 深度、进度、澄清回调、审计钩子 |

## 配置

`config.json` → `agent` 段：

| 字段 | 默认 | 说明 |
|------|------|------|
| `max_steps` | 80 | 单 turn 最大 LLM↔tool 轮数 |
| `sub_agent_max_steps` | 20 | `delegate_task` 子 Agent 回合上限（最大 40） |
| `llm.prompt_cache` | 按 provider | 显式 `cache_control` 断点；DeepSeek/Minimax 默认 true |
| `tool_max_parallel` | 4 | 单轮并行 tool 上限（最大 16） |
| `delegate_max_parallel` | 3 | `delegate_task` / `delegate_tasks` 并发上限（最大 8） |
| `tool_timeout_sec` | 120 | 单次 tool 超时秒数（最大 600） |
| `plan_gate` | true | 写操作前发出 `plan_proposed` 事件（与 ApprovalGate 并存） |
| `hooks.tool_before` / `tool_after` | — | 可选 shell 审计脚本（stdin JSON，`fail_closed` 可阻断） |
| `temperature` | 0.2 | 传给 Provider |
| `context_token_budget` | 模型相关 | 压缩阈值参考 |
| `active_profile` | `default` | 未设 `GEEGOO_PROFILE` 时使用的 profile 名 |
| `profiles` | — | 按 profile 覆盖 `output_dir` / `mcp_token` / `chat_toolsets` / `dry_run` |

### Profile 运行场景（可选）

可选：同一 `config.json` 内预设多套「情景补丁」，按需切换，无需维护多份配置文件。

优先级：`GEEGOO_PROFILE` 环境变量 > `active_profile` > `default`。

```json
{
  "active_profile": "work",
  "output_dir": "./data",
  "profiles": {
    "work": {
      "output_dir": "./data/work",
      "mcp_token": "mcp_xxx",
      "chat_toolsets": ["market"]
    },
    "sandbox": { "dry_run": true }
  }
}
```

`geegoo doctor` 仅在配置了 `profiles` 或显式选择 profile 时打印当前情景；`geegoo inspect` / `geegoo verify agent-loop` 同理。

### 运维与集成命令

```bash
# 只读汇总：profile、loop 参数、toolsets、skills（无网络）
geegoo inspect --config ~/.geegoo/config.json

# 内嵌离线 parity 卡片
geegoo inspect --quick

# Headless 单次提问 + NDJSON 事件流（CI / 脚本）
geegoo chat --message "..." --output-format ndjson --cli

# Hermes parity 验收
geegoo verify agent-loop --config ~/.geegoo/config.json
```

HTTP runtime：SSE 流含 `clarify` 事件；客户端 `POST /v1/chat/clarify` 提交 `{session_id, answer}` 或 `{skip:true}` 续跑。

压缩阈值见 `internal/prompt/compressor.go`（回合开始 85%、每轮 LLM 前 50%）。

## 可观测性

- `SetProgress(fn)` — chatui spinner / 工具预览；NDJSON 编码见 `schema_version: 1` 事件
- Hermes 对齐回调：`thinking_start/stop`、`step_complete`、`tool_gen_start/delta`、`subagent_*`、`stream_delta`、`plan_proposed`
- `delegate_task` / `delegate_tasks` — 子 Agent 独立会话 + `sub_agent_max_steps` 预算；禁止嵌套；批量并行受 `delegate_max_parallel` 限制
- **工具 schema 校验** — `Registry.Execute` 前 `ValidateArguments`（必填项与基础类型）
- **Hooks** — `tool_before` / `tool_after` shell 脚本（可选 `fail_closed`）
- **Chat 工具拦截** — interactive 模式从 schema 剔除 workflow 独占 tool；运行时 `tool_intercepted` 兜底（如 `read_working_state` → 引导 `recall`）
- **Prompt cache** — `llm.ApplyCacheBreakpoints`（system + 稳定历史边界）；`prompt_cache` 事件上报 hit/miss tokens
- `Result.Meta` — HTTP 工具 `api_code`、`duration_ms`
- EventBus（L0）— workflow 路径发 `ToolCalled` / `ToolCompleted`；chat 路径发 `TurnStarted`/`TurnCompleted`；报告合成发 `SynthesisStarted`/`SynthesisCompleted`

## 可中断性

`context.Context` 贯穿 `Gateway.Chat` 与 `Registry.Execute`。用户 Ctrl+C 中断**当前回合**；会话历史已持久化，下回合可继续。

## 测试

- `internal/agent/agent_test.go`
- `internal/agent/loop_test.go`
- `internal/agent/loop_interrupt_test.go`
- `internal/agent/loop_compress_test.go`
- `internal/verify/agent_loop_test.go` — 离线 parity 卡片

### 离线验收（Hermes parity）

```bash
geegoo verify agent-loop --config ~/.geegoo/config.json
```

检查项：`clarify` / `delegate_task` / `delegate_tasks` / `recall` / `search_code` 注册、prompt cache 断点、workflow/chat 工具隔离、子 Agent 嵌套防护、工具 schema 校验。全部 PASS 时退出码为 0。

线上验收（2026-07-19，`geegoo inspect --quick`）：9/9 PASS。

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
