"""Unit tests for pre-market workflow phase A."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from geegoo_agent.clients.market import (
    MarketClient,
    McpAnalysisResult,
    TradingDayData,
    UserBotCode,
)
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.runtime.pre_market_constants import PRE_MARKET_INDEX_CODES
from geegoo_agent.runtime.pre_market_workflow import PRE_MARKET_PHASE_A_STEPS
from geegoo_agent.runtime.session import SessionManager
from geegoo_agent.runtime.workflow import WorkflowRunner
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolContext

PROJECT_ROOT = Path(__file__).resolve().parents[2]


@pytest.fixture
def phase_a_env(tmp_path):
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
    registry = register_mvp_tools(ToolRegistry(), project_root=PROJECT_ROOT)
    executor = Executor(registry, bus)
    working_store = WorkingMemoryStore(store)
    checkpoint_mgr = CheckpointManager(store)
    runner = WorkflowRunner(executor, working_store, checkpoint_mgr, bus)
    session_mgr = SessionManager(store)
    ctx = ToolContext(
        session_id="sess-phase-a",
        mcp_token="mcp-test",
        dry_run=True,
        workspace_root=tmp_path,
        market_client=market,
        project_root=PROJECT_ROOT,
        event_bus=bus,
    )
    return {
        "runner": runner,
        "working_store": working_store,
        "checkpoint_mgr": checkpoint_mgr,
        "session_mgr": session_mgr,
        "ctx": ctx,
        "market": market,
        "tmp_path": tmp_path,
    }


@pytest.mark.unit
def test_phase_a_step_count() -> None:
    assert len(PRE_MARKET_PHASE_A_STEPS) == 11
    tools = [step.tool for step in PRE_MARKET_PHASE_A_STEPS]
    assert tools.count("get_mcp_analysis") == 5
    assert tools.count("fetch_market_news") == 3


@pytest.mark.unit
def test_non_trading_day_short_circuits_phase_a(phase_a_env) -> None:
    phase_a_env["market"].check_trading_day.return_value = TradingDayData(
        is_trading_day=False,
        date="2026-06-05",
        market="HK",
        code="00700.HK",
    )
    phase_a_env["ctx"].dry_run = False
    session = phase_a_env["session_mgr"].create("pre_market")
    working = phase_a_env["working_store"].create(session.id)
    phase_a_env["ctx"].session_id = session.id

    result = phase_a_env["runner"].run(
        session,
        PRE_MARKET_PHASE_A_STEPS,
        phase_a_env["ctx"],
        working,
    )

    assert result.ok
    assert result.working.is_trading_day is False
    assert result.working.market_context.indices_done is False
    assert session.step == 1
    phase_a_env["market"].get_report_bot_codes.assert_not_called()


@pytest.mark.unit
def test_phase_a_dry_run_marks_indices_and_news_done(phase_a_env) -> None:
    session = phase_a_env["session_mgr"].create("pre_market")
    working = phase_a_env["working_store"].create(session.id)
    phase_a_env["ctx"].session_id = session.id

    result = phase_a_env["runner"].run(
        session,
        PRE_MARKET_PHASE_A_STEPS,
        phase_a_env["ctx"],
        working,
    )

    assert result.ok
    assert result.working.market_context.indices_done is True
    assert result.working.market_context.market_news_done is True
    assert set(result.working.market_context.index_codes_done) == set(PRE_MARKET_INDEX_CODES)
    assert set(result.working.market_context.market_news.keys()) == {"US", "CN", "HK"}
    assert result.working.phase == "phase_a"
    assert session.step == len(PRE_MARKET_PHASE_A_STEPS)


@pytest.mark.unit
def test_phase_a_writes_execution_log_and_checkpoint(phase_a_env) -> None:
    session = phase_a_env["session_mgr"].create("pre_market")
    working = phase_a_env["working_store"].create(session.id)
    phase_a_env["ctx"].session_id = session.id

    phase_a_env["runner"].run(
        session,
        PRE_MARKET_PHASE_A_STEPS,
        phase_a_env["ctx"],
        working,
    )

    latest = phase_a_env["checkpoint_mgr"].load_latest(session.id)
    assert latest is not None
    assert latest.step == len(PRE_MARKET_PHASE_A_STEPS)

    logs = list(phase_a_env["tmp_path"].rglob("execution-log.md"))
    assert logs, "expected execution-log.md under workspace"
    content = logs[0].read_text(encoding="utf-8")
    assert "check_trading_day" in content
    assert "phase_a_complete" in content


@pytest.mark.unit
@patch("geegoo_agent.tools.news._run_script", return_value="headline 1")
def test_phase_a_market_news_calls_scripts(mock_run, phase_a_env) -> None:
    phase_a_env["ctx"].dry_run = False
    phase_a_env["market"].get_mcp_analysis.return_value = McpAnalysisResult(
        analysis_result="index analysis ok"
    )
    session = phase_a_env["session_mgr"].create("pre_market")
    working = phase_a_env["working_store"].create(session.id)
    phase_a_env["ctx"].session_id = session.id

    result = phase_a_env["runner"].run(
        session,
        PRE_MARKET_PHASE_A_STEPS,
        phase_a_env["ctx"],
        working,
    )

    assert result.ok
    assert mock_run.call_count == 3
    assert result.working.market_context.market_news_done is True
