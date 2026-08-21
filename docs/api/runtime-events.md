# Runtime Events (P1-2)

Thread → Turn → Item 协议：`schema_version=1`。

## Item types

| `item_type` | 典型 `event` |
|-------------|--------------|
| `user_message` | `turn_start` |
| `reasoning` | `thinking` |
| `assistant_message` | `assistant_delta`, `reply` |
| `tool_call` | `tool_start` |
| `tool_result` | `tool_end` |
| `plan_proposal` | `plan_proposed` |
| `clarify_prompt` | `clarify` |
| `budget_warning` | `budget_warning` |
| `turn_complete` | `turn_complete`, `turn_end`, `done` |

CLI NDJSON 与 HTTP SSE 共用 `internal/runtime/agent_events.go` 编码；每条事件含 `item_type` 字段。

实现：`internal/runtime/events/schema.go`。
