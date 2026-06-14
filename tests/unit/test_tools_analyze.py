"""Unit tests for analysis tools."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from geegoo_agent.clients.market import (
    BotYesterdayAttitude,
    CapitalDistributionData,
    CapitalFlowItem,
    MarketClient,
    McpAnalysisResult,
)
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext


@pytest.fixture
def ctx(tmp_path) -> ToolContext:
    market = MagicMock(spec=MarketClient)
    market.get_mcp_analysis.return_value = McpAnalysisResult(analysis_result="analysis text")
    market.get_capital_flow.return_value = [
        CapitalFlowItem(main_in_flow=-100.0, capital_flow_item_time="2026-06-05")
    ]
    market.get_capital_distribution.return_value = CapitalDistributionData(
        capital_in_super=1e9,
        capital_out_super=8e8,
        update_time="2026-06-05 15:59:59",
    )
    market.get_bot_yesterday_attitude.return_value = BotYesterdayAttitude(
        attitude="bullish",
        analysis_report="yesterday",
        bot_id="bot-1",
        found=True,
    )
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
def test_get_mcp_analysis_tool(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(
            name="get_mcp_analysis",
            arguments={
                "name": "腾讯控股",
                "code": "00700.HK",
                "period": "weekly",
            },
        ),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["analysis_result"] == "analysis text"


@pytest.mark.unit
def test_get_capital_flow_tool(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="get_capital_flow", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["latest"]["main_in_flow"] == -100.0


@pytest.mark.unit
def test_get_capital_flow_skips_a_share(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="get_capital_flow", arguments={"code": "600519.SH"}),
        ctx,
    )
    assert result.status == "skipped"
    ctx.market_client.get_capital_flow.assert_not_called()


@pytest.mark.unit
def test_get_capital_distribution_tool(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="get_capital_distribution", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert "超大单" in result.data["formatted"]


@pytest.mark.unit
def test_get_capital_distribution_skips_a_share(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="get_capital_distribution", arguments={"code": "000001.SZ"}),
        ctx,
    )
    assert result.status == "skipped"
    ctx.market_client.get_capital_distribution.assert_not_called()


@pytest.mark.unit
def test_get_bot_yesterday_attitude_maps_result(registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(
            name="get_bot_yesterday_attitude",
            arguments={"bot_id": "bot-1", "code": "00700.HK"},
        ),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["result"] == "long"


@pytest.mark.unit
def test_get_mcp_analysis_dry_run(registry: ToolRegistry, ctx: ToolContext) -> None:
    ctx.dry_run = True
    result = registry.execute(
        ToolCallRequest(
            name="get_mcp_analysis",
            arguments={"name": "腾讯", "code": "00700.HK", "period": "hourly"},
        ),
        ctx,
    )
    assert result.status == "dry_run"
    ctx.market_client.get_mcp_analysis.assert_not_called()
