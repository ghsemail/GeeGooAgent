"""Unit tests for perception tools."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from geegoo_agent.clients.market import MarketClient, TradingDayData, UserBotCode
from geegoo_agent.runtime.executor import Executor
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext


@pytest.fixture
def ctx(tmp_path) -> ToolContext:
    market = MagicMock(spec=MarketClient)
    market.check_trading_day.return_value = TradingDayData(
        is_trading_day=False,
        date="2026-06-05",
        market="HK",
        code="00700.HK",
    )
    market.get_report_bot_codes.return_value = [
        UserBotCode(
            stock_name="腾讯控股",
            code="00700.HK",
            bot_id="id-1",
            bot_name="bot",
            bot_type="DCA",
        )
    ]
    return ToolContext(
        session_id="sess-1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
    )


@pytest.fixture
def registry() -> ToolRegistry:
    return register_mvp_tools(ToolRegistry())


@pytest.mark.unit
def test_check_trading_day_calls_market_client(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="check_trading_day", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["is_trading_day"] is False
    ctx.market_client.check_trading_day.assert_called_once_with("mcp-test", "00700.HK")


@pytest.mark.unit
def test_check_trading_day_dry_run_skips_http(registry: ToolRegistry, ctx: ToolContext) -> None:
    ctx.dry_run = True
    result = registry.execute(
        ToolCallRequest(name="check_trading_day"),
        ctx,
    )
    assert result.status == "dry_run"
    ctx.market_client.check_trading_day.assert_not_called()


@pytest.mark.unit
def test_get_report_bot_codes_dry_run_returns_sample_bots(
    registry: ToolRegistry, ctx: ToolContext
) -> None:
    ctx.dry_run = True
    result = registry.execute(ToolCallRequest(name="get_report_bot_codes"), ctx)
    assert result.status == "dry_run"
    assert result.data is not None
    assert len(result.data["bots"]) == 1
    assert result.data["bots"][0]["code"] == "00700.HK"
    ctx.market_client.get_report_bot_codes.assert_not_called()


@pytest.mark.unit
def test_get_report_bot_codes_returns_bots(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(ToolCallRequest(name="get_report_bot_codes"), ctx)
    assert result.status == "ok"
    assert result.data is not None
    assert len(result.data["bots"]) == 1
    assert result.data["bots"][0]["bot_id"] == "id-1"


@pytest.mark.unit
def test_write_execution_log_creates_file(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(
            name="write_execution_log",
            arguments={"step": "check_trading_day", "message": "done", "status": "ok"},
        ),
        ctx,
    )
    assert result.status == "ok"
    from pathlib import Path

    log_file = Path(result.data["path"])
    assert log_file.exists()
    assert "check_trading_day" in log_file.read_text(encoding="utf-8")


@pytest.mark.unit
def test_executor_delegates_to_registry(ctx: ToolContext, registry: ToolRegistry) -> None:
    executor = Executor(registry)
    result = executor.execute(ToolCallRequest(name="get_report_bot_codes"), ctx)
    assert result.status == "ok"
