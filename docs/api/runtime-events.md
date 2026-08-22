# Runtime Events (P1-2)

Thread → Turn → Item 协议：`schema_version=1`。

## Payload shape (HTTP SSE + live progress)

`POST /v1/chat/stream` and `GET /v1/sessions/events/stream` progress lines use:

```json
{
  "schema_version": 1,
  "event": "gate",
  "item_type": "status",
  "ts": "2026-08-22T05:00:00.000000000Z",
  "decision": "retrieve",
  "hits": 2,
  "data": { "decision": "retrieve", "hits": 2 }
}
```

- **Legacy clients** may keep reading flat fields (`decision`, `hits`, …).
- **Structured clients** should use `item_type` + nested `data`.
- SSE envelope: `event: gate` with the JSON above in `data:`.

CLI NDJSON uses the same fields via `runtime.ProgressToAgentEvent`.

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
| `status` | `gate`, `status`, `context_fragment_applied` |
| `turn_complete` | `turn_complete`, `turn_end`, `done` |

## Context fragments (P1-1)

When recall + procedural skills inject, the loop emits:

```json
{ "event": "context_fragment_applied", "applied": ["recall", "procedural"], "dropped": [], "bytes": 1234 }
```

Implementation: `internal/context/`, `internal/agent/dynamic_context.go`.
