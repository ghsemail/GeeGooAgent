# L4 — WorkflowEngine

## 职责

Run 级编排：加载 Skill → 启动 Loop → 触发 Supervisor → 处理补跑。

## 入口

**路径**：`src/geegoo/runtime/agent_runtime.py`

```python
class AgentRuntime:
    def __init__(
        self,
        skill_loader: SkillLoader,
        loop: ReActLoop,
        supervisor: SupervisorEngine,
        bus: EventBus,
        state_store: StateStore,
    ): ...

    def run(self, ctx: RunContext) -> RunResult:
        skill = self.skill_loader.load(ctx.skill_name, ctx.mode)
        session = self._init_or_resume_session(ctx)
        self.bus.emit("RunStarted", {"session_id": session.id, "skill": skill.name})

        loop_result = self.loop.run(session, skill)
        sup_result = self.supervisor.verify(session, loop_result.working, skill.checks)

        if sup_result.needs_retry:
            return self._retry_missing(session, skill, sup_result)
        if not sup_result.passed:
            session.fail(sup_result.summary)
            self.bus.emit("RunFailed", {"session_id": session.id})
            return RunResult.failed(sup_result)

        session.finalize("completed")
        self.bus.emit("RunFinished", {"session_id": session.id, "status": "completed"})
        return RunResult.ok(session)
```

## 与 Supervisor 协作

见 [cross-cutting/supervisor.md](../../cross-cutting/supervisor.md)。

## 非交易日短路

由 Agent 调 `check_trading_day` Tool；WorkflowEngine 不硬编码——但若 Session 在 step 0 已标记 `is_trading_day=false`，Supervisor 应接受「无报告」为合法完成。

## MVP

`run(pre_market)` + Supervisor + 单次补跑 resume。