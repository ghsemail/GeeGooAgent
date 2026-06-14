"""Workflow runner — step table driven execution."""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

from geegoo_agent.exceptions import GeeGooAgentError
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.runtime.session import Session
from geegoo_agent.tools.types import ToolCallRequest, ToolContext, ToolResult


@dataclass
class WorkflowStep:
    name: str
    tool: str
    arguments: dict[str, Any] | Callable[[PreMarketWorking], dict[str, Any]] = field(
        default_factory=dict
    )


@dataclass
class RunResult:
    session_id: str
    status: str
    working: PreMarketWorking
    last_error: str | None = None

    @property
    def ok(self) -> bool:
        return self.status == "completed"


PRE_MARKET_STUB_STEPS: list[WorkflowStep] = [
    WorkflowStep("check_trading_day", "check_trading_day", {"code": "00700.HK"}),
    WorkflowStep("get_report_bot_codes", "get_report_bot_codes", {}),
    WorkflowStep(
        "log_complete",
        "write_execution_log",
        lambda _w: {
            "step": "pre_market_stub",
            "message": "MVP workflow stub finished",
            "status": "ok",
        },
    ),
]


class WorkflowRunner:
    def __init__(
        self,
        executor: Executor,
        working_store: WorkingMemoryStore,
        checkpoint_mgr: CheckpointManager,
        event_bus: InProcessEventBus,
    ) -> None:
        self._executor = executor
        self._working_store = working_store
        self._checkpoint_mgr = checkpoint_mgr
        self._bus = event_bus

    def _write_execution_log(self, ctx: ToolContext, step_name: str, result: ToolResult) -> None:
        log_status = result.status if result.status in {"ok", "error", "skipped"} else "ok"
        self._executor.execute(
            ToolCallRequest(
                name="write_execution_log",
                arguments={
                    "step": step_name,
                    "message": result.summary[:500],
                    "status": log_status,
                },
            ),
            ctx,
        )

    def _resolve_arguments(
        self,
        step: WorkflowStep,
        working: PreMarketWorking,
    ) -> dict[str, Any]:
        if callable(step.arguments):
            return step.arguments(working)
        return step.arguments

    def _checkpoint(
        self,
        session: Session,
        working: PreMarketWorking,
        step_index: int,
        tool_name: str,
    ) -> None:
        self._checkpoint_mgr.save(
            session_id=session.id,
            step=step_index,
            skill=session.skill_name,
            status=session.status,
            working=working.model_dump(),
            last_tool=tool_name,
        )

    def _process_step(
        self,
        session: Session,
        step: WorkflowStep,
        ctx: ToolContext,
        working: PreMarketWorking,
        step_index: int,
    ) -> tuple[PreMarketWorking, ToolResult | None]:
        arguments = self._resolve_arguments(step, working)
        try:
            result = self._executor.execute(
                ToolCallRequest(name=step.tool, arguments=arguments),
                ctx,
            )
        except GeeGooAgentError as exc:
            session.mark_failed(str(exc))
            self._bus.emit("RunFailed", {"session_id": session.id, "error": str(exc)})
            return working, None

        working = self._working_store.apply(working, step.tool, result)

        if (
            step.tool == "get_bot_yesterday_attitude"
            and ctx.llm_gateway is not None
            and not ctx.dry_run
            and working.current_stock_code
        ):
            from geegoo_agent.runtime.llm_tasks import enrich_stock_with_llm

            working = enrich_stock_with_llm(
                ctx.llm_gateway,
                working,
                working.current_stock_code,
                session_id=ctx.session_id,
                step=ctx.step,
            )
            self._working_store.save(working)

        session.step = step_index

        if step.tool != "write_execution_log":
            self._write_execution_log(ctx, step.name, result)

        self._checkpoint(session, working, step_index, step.tool)

        if result.status == "error":
            session.mark_failed(result.summary)
            self._bus.emit("RunFailed", {"session_id": session.id, "error": result.summary})
            return working, result

        if step.tool == "check_trading_day" and working.is_trading_day is False:
            session.mark_completed()
            self._bus.emit(
                "RunFinished",
                {"session_id": session.id, "status": "completed", "reason": "non_trading_day"},
            )
            return working, result

        return working, result

    def _run_flat_steps(
        self,
        session: Session,
        steps: list[WorkflowStep],
        ctx: ToolContext,
        working: PreMarketWorking,
        start_index: int,
    ) -> tuple[PreMarketWorking, RunResult | None]:
        for index in range(start_index, len(steps)):
            step = steps[index]
            ctx.step = index
            working, result = self._process_step(session, step, ctx, working, index + 1)
            if result is None:
                return working, RunResult(session.id, "failed", working, session.error)
            if step.tool == "check_trading_day" and working.is_trading_day is False:
                return working, RunResult(session.id, "completed", working)
            if result.status == "error":
                return working, RunResult(session.id, "failed", working, result.summary)
        return working, None

    def _run_per_stock_steps(
        self,
        session: Session,
        steps: list[WorkflowStep],
        ctx: ToolContext,
        working: PreMarketWorking,
        flat_step_count: int,
    ) -> tuple[PreMarketWorking, RunResult | None]:
        working = working.model_copy(deep=True)
        working.phase = "phase_b"
        self._working_store.save(working)

        step_counter = flat_step_count
        for bot in working.bot_codes:
            code = bot.code
            ws = working.stocks.get(code)
            if ws is None or ws.status in {"reported", "skipped"}:
                continue

            working = working.model_copy(deep=True)
            working.current_stock_code = code
            if working.stocks[code].status == "pending":
                working.stocks[code].status = "collecting"
            self._working_store.save(working)

            skip_stock = False
            for step in steps:
                if skip_stock:
                    break
                step_counter += 1
                ctx.step = step_counter
                step_name = f"{code}/{step.name}"
                working, result = self._process_step(
                    session,
                    WorkflowStep(step_name, step.tool, step.arguments),
                    ctx,
                    working,
                    step_counter,
                )
                if result is None:
                    return working, RunResult(session.id, "failed", working, session.error)
                if result.status == "error":
                    working.stocks[code].status = "failed"
                    self._working_store.save(working)
                    return working, RunResult(session.id, "failed", working, result.summary)
                if (
                    step.tool == "list_today_reports"
                    and result.data
                    and result.data.get("already_reported")
                ):
                    skip_stock = True

        working = working.model_copy(deep=True)
        working.current_stock_code = None
        if all(
            working.stocks[c].status in {"reported", "skipped"}
            for c in working.stocks
        ):
            working.phase = "done"
        self._working_store.save(working)
        return working, None

    def run(
        self,
        session: Session,
        steps: list[WorkflowStep],
        ctx: ToolContext,
        working: PreMarketWorking,
        *,
        per_stock_steps: list[WorkflowStep] | None = None,
        start_index: int = 0,
    ) -> RunResult:
        session.mark_running()
        flat_len = len(steps)

        if working.is_trading_day is not False and start_index < flat_len:
            working, early = self._run_flat_steps(session, steps, ctx, working, start_index)
            if early is not None:
                return early

        if per_stock_steps and working.is_trading_day is not False:
            working, early = self._run_per_stock_steps(
                session,
                per_stock_steps,
                ctx,
                working,
                flat_len if start_index < flat_len else start_index,
            )
            if early is not None:
                return early

        session.mark_completed()
        self._bus.emit("RunFinished", {"session_id": session.id, "status": "completed"})
        return RunResult(session.id, "completed", working)
