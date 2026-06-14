"""Unit tests for WorkingMemoryStore."""

from __future__ import annotations

import pytest

from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.models import StockWorkspace
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.tools.types import ToolResult


@pytest.fixture
def working_store(tmp_path) -> WorkingMemoryStore:
    return WorkingMemoryStore(FileStateStore(tmp_path))


@pytest.mark.unit
def test_create_and_load_roundtrip(working_store: WorkingMemoryStore) -> None:
    created = working_store.create("sess-1")
    loaded = working_store.load("sess-1")
    assert loaded is not None
    assert loaded.session_id == created.session_id
    assert loaded.phase == "init"


@pytest.mark.unit
def test_apply_check_trading_day_sets_phase_a(working_store: WorkingMemoryStore) -> None:
    working = working_store.create("sess-2")
    updated = working_store.apply(
        working,
        "check_trading_day",
        ToolResult(
            status="ok",
            summary="ok",
            data={"is_trading_day": True, "code": "00700.HK", "market": "HK", "date": "2026-06-05"},
        ),
    )
    assert updated.is_trading_day is True
    assert updated.phase == "phase_a"
    reloaded = working_store.load("sess-2")
    assert reloaded is not None
    assert reloaded.is_trading_day is True


@pytest.mark.unit
def test_apply_get_report_bot_codes_initializes_stocks(working_store: WorkingMemoryStore) -> None:
    working = working_store.create("sess-3")
    working.phase = "phase_a"
    updated = working_store.apply(
        working,
        "get_report_bot_codes",
        ToolResult(
            status="ok",
            summary="ok",
            data={
                "bots": [
                    {
                        "stock_name": "腾讯控股",
                        "code": "00700.HK",
                        "bot_id": "b1",
                        "bot_name": "DCA",
                        "bot_type": "DCA",
                    }
                ]
            },
        ),
    )
    assert len(updated.bot_codes) == 1
    assert "00700.HK" in updated.stocks
    assert updated.stocks["00700.HK"].stock_name == "腾讯控股"
    assert updated.phase == "phase_a"


@pytest.mark.unit
def test_summary_includes_phase_and_pending(working_store: WorkingMemoryStore) -> None:
    working = working_store.create("sess-4")
    working.phase = "phase_b"
    working.is_trading_day = True
    working.bot_codes = []
    working.stocks = {"00700.HK": StockWorkspace(code="00700.HK", status="collecting")}
    text = working_store.summary(working)
    assert "phase=phase_b" in text
    assert "00700.HK(collecting)" in text


@pytest.mark.unit
def test_apply_get_mcp_analysis_marks_indices_done(working_store: WorkingMemoryStore) -> None:
    working = working_store.create("sess-6")
    working.phase = "phase_a"
    for code in ["^DJI.US", "^IXIC.US", "000001.SH", "399001.SZ", "800000.HK"]:
        working = working_store.apply(
            working,
            "get_mcp_analysis",
            ToolResult(
                status="ok",
                summary="ok",
                data={"code": code, "analysis_result": f"analysis-{code}"},
            ),
        )
    assert working.market_context.indices_done is True
    assert len(working.market_context.index_codes_done) == 5


@pytest.mark.unit
def test_apply_fetch_market_news_marks_news_done(working_store: WorkingMemoryStore) -> None:
    working = working_store.create("sess-7")
    for market in ["US", "CN", "HK"]:
        working = working_store.apply(
            working,
            "fetch_market_news",
            ToolResult(
                status="ok",
                summary="ok",
                data={"market": market, "text": f"news-{market}"},
            ),
        )
    assert working.market_context.market_news_done is True


@pytest.mark.unit
def test_non_trading_day_sets_done(working_store: WorkingMemoryStore) -> None:
    working = working_store.create("sess-5")
    updated = working_store.apply(
        working,
        "check_trading_day",
        ToolResult(status="ok", summary="skip", data={"is_trading_day": False}),
    )
    assert updated.phase == "done"
