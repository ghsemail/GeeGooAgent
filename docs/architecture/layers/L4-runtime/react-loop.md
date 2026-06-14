# L4 — ReAct Loop

## 职责

显式 **Observe → Plan → Act → Update** 循环，直到完成或 `max_steps`。

## 伪代码

```python
class ReActLoop:
    def run(self, session: Session, skill: LoadedSkill) -> SessionResult:
        working = self.state_store.load_working(session.id)

        while session.status == "running":
            if session.step >= self.config.max_steps:
                session.fail("max_steps exceeded")
                break

            # OBSERVE
            messages = self.context_builder.build(session, working)

            # PLAN
            response = self.gateway.chat(messages, tools=self.tools.schemas(skill))
            session.append_assistant(response)
            self.bus.emit("PlanCreated", {"step": session.step})

            if not response.tool_calls:
                session.mark_agent_done()
                break

            # ACT
            for call in response.tool_calls:
                self.bus.emit("ToolCalled", {"tool": call.name, "step": session.step})
                result = self.executor.execute(call, session, working)
                working.apply(call, result)
                session.append_tool_result(call, result.summary)
                self.bus.emit("ToolCompleted", {"tool": call.name, "status": result.status})

            session.step += 1
            self.checkpoint.save(session, working)
            self.context_builder.compact_if_needed(session)

        return SessionResult(session=session, working=working)
```

## 配置

```json
{
  "agent": {
    "max_steps": 80,
    "temperature": 0.2,
    "context_token_budget": 80000,
    "compact_after_steps": 15
  }
}
```

## StepRecord（可观测）

每步写入 `session.step_records[]`：

```json
{
  "step": 12,
  "timestamp": "ISO8601",
  "phase": "act",
  "tool_calls": [{"name": "get_mcp_analysis", "args": {}}],
  "tool_status": ["ok"],
  "tokens_used": 4200,
  "latency_ms": 3100
}
```

## MVP

完整 Loop + dry-run 分支（create_report mock）。