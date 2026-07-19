# HTTP Runtime — Clarify 两阶段协议

> 更新：2026-07-19。配合 `internal/runtimeapi/clarify_hub.go` 与 Agent `clarify` 工具。

## 流程

```text
POST /v1/chat/completions (stream=true)
  → Agent 调用 clarify 工具
  → SSE 推送 geegoo.agent_event: { "event": "clarify", "session_id", "question", "choices" }
  → 客户端 POST /v1/chat/clarify
  → 同一 session 内 Agent 继续执行直至 turn 结束
```

## POST /v1/chat/clarify

```json
{
  "session_id": "api-150405",
  "answer": "A股"
}
```

跳过澄清：

```json
{ "session_id": "api-150405", "skip": true }
```

响应：`{"status":"ok"}`；无 pending clarify 时返回 404。

## SSE 事件格式

```json
{
  "object": "geegoo.agent_event",
  "data": {
    "event": "clarify",
    "session_id": "api-150405",
    "question": "选择市场",
    "choices": ["A股", "港股"]
  }
}
```

## 写操作 Plan 门控（chat）

当 `plan_gate=true`（默认）且模型提议 mutating 工具时：

1. 发出 `plan_proposed` 事件（含工具名与参数摘要）
2. **不执行**写操作 HTTP 请求
3. 挂起的 tool_calls 写入 session `metadata.pending_plan`（跨请求可恢复）
4. 用户输入 `y`/`确认` 后执行；`n`/`取消` 放弃

### HTTP 确认（`POST /v1/chat/plan`）

在 `POST /v1/chat/stream` 返回 `turn_end.plan_pending=true` 后：

```json
{ "session_id": "<id>", "approve": true }
```

或拒绝：

```json
{ "session_id": "<id>", "reject": true }
```

响应含 `assistant_text`、`plan_pending`（应为 false）、`step_count`。

也可在下一轮流式请求中直接发送消息 `y` / `n`（与 TUI 相同）。

详见 [agent-loop.md](./agent-loop.md)。

## 验收

- clarify 单测：`internal/runtimeapi/clarify_hub_test.go`
- clarify E2E：`internal/runtimeapi/clarify_e2e_test.go`
- plan 单测：`internal/runtimeapi/plan_test.go`
