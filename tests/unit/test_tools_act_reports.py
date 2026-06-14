"""Unit tests for report action tools."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from geegoo_agent.clients.geegoo_bot import DailyReportsData, GeeGooBotClient
from geegoo_agent.clients.market import MarketClient, PreMarketReportResult
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext


@pytest.fixture
def ctx(tmp_path) -> ToolContext:
    market = MagicMock(spec=MarketClient)
    market.create_pre_market_report.return_value = PreMarketReportResult(
        report_id="report-123"
    )
    geegoo = MagicMock(spec=GeeGooBotClient)
    geegoo.get_stock_daily_reports.return_value = DailyReportsData(
        pre_market=[{"report_id": "existing"}],
        intraday=[],
        post_market=[],
    )
    return ToolContext(
        session_id="sess-1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
        geegoo_bot_client=geegoo,
    )


@pytest.fixture
def registry() -> ToolRegistry:
    return register_mvp_tools(ToolRegistry())


def _report_args() -> dict:
    return {
        "code": "00700.HK",
        "stock_name": "腾讯控股",
        "bot_id": "bot-1",
        "bot_name": "DCA",
        "bot_type": "DCA",
        "result": "long",
        "confidence": "high",
        "reason": "test reason",
        "suggestion": "buy",
        "report": "full report",
    }


@pytest.mark.unit
def test_create_pre_market_report_validates_and_calls_api(
    registry: ToolRegistry,
    ctx: ToolContext,
) -> None:
    result = registry.execute(
        ToolCallRequest(name="create_pre_market_report", arguments=_report_args()),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["report_id"] == "report-123"
    ctx.market_client.create_pre_market_report.assert_called_once()


@pytest.mark.unit
def test_create_pre_market_report_rejects_empty_bot_id(
    registry: ToolRegistry,
    ctx: ToolContext,
) -> None:
    args = _report_args()
    args["bot_id"] = ""
    result = registry.execute(
        ToolCallRequest(name="create_pre_market_report", arguments=args),
        ctx,
    )
    assert result.status == "error"
    ctx.market_client.create_pre_market_report.assert_not_called()


@pytest.mark.unit
def test_save_local_report_writes_file(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(
            name="save_local_report",
            arguments={"code": "00700.HK", "content": "# report"},
        ),
        ctx,
    )
    assert result.status == "ok"
    from pathlib import Path

    path = Path(result.data["path"])
    assert path.exists()
    assert path.read_text(encoding="utf-8") == "# report"


@pytest.mark.unit
def test_get_stock_daily_reports_tool(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(
            name="get_stock_daily_reports",
            arguments={"code": "00700.HK", "report_date": "2026-06-05"},
        ),
        ctx,
    )
    assert result.status == "ok"
    assert len(result.data["pre_market"]) == 1


@pytest.mark.unit
def test_list_today_reports_detects_existing(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="list_today_reports", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data["already_reported"] is True
