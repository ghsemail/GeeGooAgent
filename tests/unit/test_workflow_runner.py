"""Unit tests for WorkflowRunner and Session."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from geegoo_agent.clients.market import MarketClient, TradingDayData, UserBotCode
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.runtime.session import SessionManager
from geegoo_agent.runtime.workflow import PRE_MARKET_STUB_STEPS, WorkflowRunner, WorkflowStep
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolContext


@pytest.fixture
def workflow_env(tmp_path):
    store = FileStateStore(tmp_path)
    bus = InProcessEventBus()
    market = MagicMock(spec=MarketClient)
    market.check_trading_day.return_value = TradingDayData(
        is_trading_day=True,
        date="2026-06-05",
        market="HK",
        code="00700.HK",
    )
    market.get_report_bot_codes.return_value = [
        UserBotCode(
            stock_name="腾讯控股",
            code="00700.HK",
            bot_id="bot-1",
            bot_name="test-bot",
            bot_type="DCA",
        )
    ]
    registry = register_mvp_tools(ToolRegistry())
    executor = Executor(registry, bus)
    working_store = WorkingMemoryStore(store)
    checkpoint_mgr = CheckpointManager(store)
    runner = WorkflowRunner(executor, working_store, checkpoint_mgr, bus)
    session_mgr = SessionManager(store)
    ctx = ToolContext(
        session_id="sess-test",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
        event_bus=bus,
    )
    return {
        "runner": runner,
        "working_store": working_store,
        "checkpoint_mgr": checkpoint_mgr,
        "session_mgr": session_mgr,
        "ctx": ctx,
        "market": market,
        "bus": bus,
    }


@pytest.mark.unit
def test_session_create_and_load(workflow_env) -> None:
    session = workflow_env["session_mgr"].create("pre_market")
    loaded = workflow_env["session_mgr"].load(session.id)
    assert loaded is not None
    assert loaded.skill_name == "pre_market"
    assert loaded.status == "created"


@pytest.mark.unit
def test_stub_workflow_completes_all_steps(workflow_env) -> None:
    session = workflow_env["session_mgr"].create("pre_market")
    working = workflow_env["working_store"].create(session.id, skill="pre_market")
    workflow_env["ctx"].session_id = session.id

    result = workflow_env["runner"].run(
        session,
        PRE_MARKET_STUB_STEPS,
        workflow_env["ctx"],
        working,
    )

    assert result.ok
    assert session.status == "completed"
    assert session.step == 3
    assert result.working.is_trading_day is True
    assert len(result.working.bot_codes) == 1
    assert "execution_log" in result.working.artifacts
    latest = workflow_env["checkpoint_mgr"].load_latest(session.id)
    assert latest is not None
    assert latest.step == 3
    assert latest.last_tool == "write_execution_log"


@pytest.mark.unit
def test_non_trading_day_short_circuits(workflow_env) -> None:
    workflow_env["market"].check_trading_day.return_value = TradingDayData(
        is_trading_day=False,
        date="2026-06-05",
        market="HK",
        code="00700.HK",
    )
    session = workflow_env["session_mgr"].create("pre_market")
    working = workflow_env["working_store"].create(session.id)
    workflow_env["ctx"].session_id = session.id

    result = workflow_env["runner"].run(
        session,
        PRE_MARKET_STUB_STEPS,
        workflow_env["ctx"],
        working,
    )

    assert result.ok
    assert result.working.is_trading_day is False
    assert result.working.phase == "done"
    assert session.step == 1
    workflow_env["market"].get_report_bot_codes.assert_not_called()


@pytest.mark.unit
def test_unknown_tool_marks_session_failed(workflow_env) -> None:
    session = workflow_env["session_mgr"].create("pre_market")
    working = workflow_env["working_store"].create(session.id)
    workflow_env["ctx"].session_id = session.id
    bad_steps = [WorkflowStep("bad", "missing_tool", {})]

    result = workflow_env["runner"].run(
        session,
        bad_steps,
        workflow_env["ctx"],
        working,
    )

    assert result.status == "failed"
    assert "unknown tool" in (result.last_error or "")
    assert session.status == "failed"


@pytest.mark.unit
def test_resume_continues_from_checkpoint(workflow_env) -> None:
    session = workflow_env["session_mgr"].create("pre_market")
    working = workflow_env["working_store"].create(session.id)
    workflow_env["ctx"].session_id = session.id

    first = workflow_env["runner"].run(
        session,
        PRE_MARKET_STUB_STEPS[:1],
        workflow_env["ctx"],
        working,
    )
    assert first.ok
    assert session.step == 1

    resumed_working = workflow_env["working_store"].load(session.id)
    assert resumed_working is not None
    checkpoint = workflow_env["checkpoint_mgr"].load_latest(session.id)
    assert checkpoint is not None

    result = workflow_env["runner"].run(
        session,
        PRE_MARKET_STUB_STEPS,
        workflow_env["ctx"],
        resumed_working,
        start_index=checkpoint.step,
    )

    assert result.ok
    assert session.step == 3
    assert len(result.working.bot_codes) == 1
