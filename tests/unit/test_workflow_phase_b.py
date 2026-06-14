"""Unit tests for pre-market workflow phase B (per-stock loop)."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from geegoo_agent.clients.geegoo_bot import DailyReportsData, GeeGooBotClient
from geegoo_agent.clients.market import (
    BotYesterdayAttitude,
    CapitalDistributionData,
    CapitalFlowItem,
    MarketClient,
    McpAnalysisResult,
    PreMarketReportResult,
    TradingDayData,
    UserBotCode,
)
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.models import BotStock, PreMarketWorking, StockWorkspace
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.runtime.pre_market_workflow import PRE_MARKET_PER_STOCK_STEPS
from geegoo_agent.runtime.session import SessionManager
from geegoo_agent.runtime.workflow import WorkflowRunner
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolContext

PROJECT_ROOT = Path(__file__).resolve().parents[2]

SAMPLE_BOT = UserBotCode(
    stock_name="腾讯控股",
    code="00700.HK",
    bot_id="bot-1",
    bot_name="test-bot",
    bot_type="DCA",
)


@pytest.fixture
def phase_b_env(tmp_path):
    store = FileStateStore(tmp_path)
    bus = InProcessEventBus()
    market = MagicMock(spec=MarketClient)
    market.check_trading_day.return_value = TradingDayData(
        is_trading_day=True,
        date="2026-06-05",
        market="HK",
        code="00700.HK",
    )
    market.get_report_bot_codes.return_value = [SAMPLE_BOT]
    market.get_capital_flow.return_value = [
        CapitalFlowItem(main_in_flow=-100.0, capital_flow_item_time="2026-06-05")
    ]
    market.get_capital_distribution.return_value = CapitalDistributionData(
        capital_in_super=1e9,
        capital_out_super=8e8,
        update_time="2026-06-05 15:59:59",
    )
    market.get_mcp_analysis.return_value = McpAnalysisResult(
        analysis_result="weekly support 300 resistance 400"
    )
    market.get_bot_yesterday_attitude.return_value = BotYesterdayAttitude(
        attitude="bullish",
        analysis_report="yesterday bullish",
        bot_id="bot-1",
        code="00700.HK",
        found=True,
    )
    market.create_pre_market_report.return_value = PreMarketReportResult(report_id="rid-1")

    geegoo = MagicMock(spec=GeeGooBotClient)
    geegoo.get_stock_daily_reports.return_value = DailyReportsData(
        pre_market=[],
        intraday=[],
        post_market=[],
    )

    registry = register_mvp_tools(ToolRegistry(), project_root=PROJECT_ROOT)
    executor = Executor(registry, bus)
    working_store = WorkingMemoryStore(store)
    checkpoint_mgr = CheckpointManager(store)
    runner = WorkflowRunner(executor, working_store, checkpoint_mgr, bus)
    session_mgr = SessionManager(store)
    ctx = ToolContext(
        session_id="sess-phase-b",
        mcp_token="mcp-test",
        dry_run=True,
        workspace_root=tmp_path,
        market_client=market,
        geegoo_bot_client=geegoo,
        project_root=PROJECT_ROOT,
        event_bus=bus,
    )
    return {
        "runner": runner,
        "working_store": working_store,
        "session_mgr": session_mgr,
        "ctx": ctx,
        "market": market,
        "geegoo": geegoo,
        "tmp_path": tmp_path,
    }


def _working_with_bot(session_id: str) -> PreMarketWorking:
    working = PreMarketWorking(session_id=session_id, skill="pre_market", phase="phase_b")
    working.is_trading_day = True
    working.bot_codes = [BotStock.model_validate(SAMPLE_BOT.model_dump())]
    working.stocks["00700.HK"] = StockWorkspace(
        code="00700.HK",
        stock_name="腾讯控股",
        bot_id="bot-1",
        bot_name="test-bot",
        bot_type="DCA",
    )
    return working


@pytest.mark.unit
def test_per_stock_step_count() -> None:
    assert len(PRE_MARKET_PER_STOCK_STEPS) == 9
    tools = [s.tool for s in PRE_MARKET_PER_STOCK_STEPS]
    assert tools[0] == "list_today_reports"
    assert "create_pre_market_report" in tools


@pytest.mark.unit
def test_single_stock_dry_run_completes(phase_b_env) -> None:
    session = phase_b_env["session_mgr"].create("pre_market")
    working = _working_with_bot(session.id)
    phase_b_env["working_store"].save(working)
    phase_b_env["ctx"].session_id = session.id

    result = phase_b_env["runner"].run(
        session,
        [],
        phase_b_env["ctx"],
        working,
        per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
    )

    assert result.ok
    stock = result.working.stocks["00700.HK"]
    assert stock.status == "reported"
    assert stock.report_id == "dry-run-id"
    assert stock.weekly_analysis_ref
    assert stock.attitude == "neutral"
    assert result.working.phase == "done"


@pytest.mark.unit
def test_single_stock_live_api_happy_path(phase_b_env) -> None:
    phase_b_env["ctx"].dry_run = False
    session = phase_b_env["session_mgr"].create("pre_market")
    working = _working_with_bot(session.id)
    phase_b_env["working_store"].save(working)
    phase_b_env["ctx"].session_id = session.id

    with patch("geegoo_agent.tools.news._run_script", return_value="stock headline"):
        result = phase_b_env["runner"].run(
            session,
            [],
            phase_b_env["ctx"],
            working,
            per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
        )

    assert result.ok
    stock = result.working.stocks["00700.HK"]
    assert stock.status == "reported"
    assert stock.report_id == "rid-1"
    assert stock.attitude == "bullish"
    phase_b_env["market"].create_pre_market_report.assert_called_once()
    saved = list(phase_b_env["tmp_path"].rglob("*-premarket.md"))
    assert saved


@pytest.mark.unit
def test_attitude_404_maps_neutral_still_reports(phase_b_env) -> None:
    phase_b_env["ctx"].dry_run = False
    phase_b_env["market"].get_bot_yesterday_attitude.return_value = (
        BotYesterdayAttitude.neutral_default("bot-1")
    )
    session = phase_b_env["session_mgr"].create("pre_market")
    working = _working_with_bot(session.id)
    phase_b_env["ctx"].session_id = session.id

    with patch("geegoo_agent.tools.news._run_script", return_value="news"):
        result = phase_b_env["runner"].run(
            session,
            [],
            phase_b_env["ctx"],
            working,
            per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
        )

    assert result.ok
    assert result.working.stocks["00700.HK"].attitude == "neutral"
    assert result.working.stocks["00700.HK"].status == "reported"


@pytest.mark.unit
def test_idempotency_skips_already_reported_stock(phase_b_env) -> None:
    phase_b_env["ctx"].dry_run = False
    phase_b_env["geegoo"].get_stock_daily_reports.return_value = DailyReportsData(
        pre_market=[{"report_id": "existing"}],
        intraday=[],
        post_market=[],
    )
    session = phase_b_env["session_mgr"].create("pre_market")
    working = _working_with_bot(session.id)
    phase_b_env["ctx"].session_id = session.id

    result = phase_b_env["runner"].run(
        session,
        [],
        phase_b_env["ctx"],
        working,
        per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
    )

    assert result.ok
    assert result.working.stocks["00700.HK"].status == "skipped"
    phase_b_env["market"].create_pre_market_report.assert_not_called()


@pytest.mark.unit
def test_two_stocks_both_reported(phase_b_env) -> None:
    session = phase_b_env["session_mgr"].create("pre_market")
    working = _working_with_bot(session.id)
    second = BotStock(
        code="AAPL.US",
        stock_name="Apple Inc.",
        bot_id="bot-2",
        bot_name="apple-bot",
        bot_type="DCA",
    )
    working.bot_codes.append(second)
    working.stocks["AAPL.US"] = StockWorkspace(
        code="AAPL.US",
        stock_name="Apple Inc.",
        bot_id="bot-2",
        bot_name="apple-bot",
        bot_type="DCA",
    )
    phase_b_env["ctx"].session_id = session.id

    result = phase_b_env["runner"].run(
        session,
        [],
        phase_b_env["ctx"],
        working,
        per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
    )

    assert result.ok
    assert result.working.stocks["00700.HK"].status == "reported"
    assert result.working.stocks["AAPL.US"].status == "reported"


@pytest.mark.unit
def test_create_report_includes_bot_fields(phase_b_env) -> None:
    phase_b_env["ctx"].dry_run = False
    session = phase_b_env["session_mgr"].create("pre_market")
    working = _working_with_bot(session.id)
    phase_b_env["ctx"].session_id = session.id

    with patch("geegoo_agent.tools.news._run_script", return_value="news"):
        phase_b_env["runner"].run(
            session,
            [],
            phase_b_env["ctx"],
            working,
            per_stock_steps=PRE_MARKET_PER_STOCK_STEPS,
        )

    call_args = phase_b_env["market"].create_pre_market_report.call_args
    body = call_args[0][1]
    assert body["bot_id"] == "bot-1"
    assert body["bot_name"] == "test-bot"
    assert body["bot_type"] == "DCA"
    assert body["stock_name"] == "腾讯控股"
