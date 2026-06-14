"""Unit tests for InProcessEventBus."""

from __future__ import annotations

import pytest

from geegoo_agent.infra.events import InProcessEventBus


@pytest.mark.unit
def test_emit_invokes_registered_handler() -> None:
    bus = InProcessEventBus()
    seen: list[dict] = []

    bus.on("ToolCompleted", lambda payload: seen.append(payload))
    bus.emit("ToolCompleted", {"tool": "check_trading_day", "status": "ok"})

    assert len(seen) == 1
    assert seen[0]["tool"] == "check_trading_day"


@pytest.mark.unit
def test_emit_records_history() -> None:
    bus = InProcessEventBus()
    bus.emit("RunStarted", {"session_id": "sess-1"})
    assert bus.history == [("RunStarted", {"session_id": "sess-1"})]


@pytest.mark.unit
def test_handler_exception_does_not_break_other_handlers() -> None:
    bus = InProcessEventBus()
    results: list[str] = []

    def bad(_: dict) -> None:
        raise RuntimeError("boom")

    bus.on("ToolCalled", bad)
    bus.on("ToolCalled", lambda _: results.append("ok"))

    bus.emit("ToolCalled", {"tool": "x"})

    assert results == ["ok"]
