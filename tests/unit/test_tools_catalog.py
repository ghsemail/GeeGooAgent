"""Catalog and HTTP API tool registration."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from geegoo_agent.clients.geegoo_bot import GeeGooBotClient
from geegoo_agent.clients.market import MarketClient
from geegoo_agent.tools.bootstrap import all_tool_instances, register_all_tools, register_mvp_tools
from geegoo_agent.tools.catalog import (
    BESPOKE_TOOL_NAMES,
    CATALOG_BY_CATEGORY,
    HTTP_TOOL_CATALOG,
    catalog_http_specs,
)
from geegoo_agent.tools.registry import BOT_MUTATION_TOOLS, ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolCategory, ToolContext


@pytest.mark.unit
def test_catalog_has_unique_tool_names() -> None:
    names = [spec.name for spec in HTTP_TOOL_CATALOG]
    assert len(names) == len(set(names))


@pytest.mark.unit
def test_catalog_excludes_bespoke_overlaps() -> None:
    http_names = {spec.name for spec in catalog_http_specs()}
    assert not http_names & BESPOKE_TOOL_NAMES


@pytest.mark.unit
def test_register_all_tools_count() -> None:
    registry = register_all_tools(ToolRegistry())
    assert len(registry.list_names()) == len(all_tool_instances())
    assert len(registry.list_names()) >= 70


@pytest.mark.unit
def test_register_mvp_tools_still_filters_manifest() -> None:
    from pathlib import Path

    project_root = Path(__file__).resolve().parents[2]
    registry = register_mvp_tools(ToolRegistry(), project_root=project_root)
    assert len(registry.list_names()) == 16


@pytest.mark.unit
def test_catalog_categories_cover_http_tools() -> None:
    categorized = {spec.name for specs in CATALOG_BY_CATEGORY.values() for spec in specs}
    assert categorized == {spec.name for spec in HTTP_TOOL_CATALOG}


@pytest.mark.unit
def test_http_api_tool_calls_geegoo_client(tmp_path) -> None:
    registry = register_all_tools(ToolRegistry())
    geegoo = MagicMock()
    geegoo.post_direct.return_value = [
        {"code": "00700.HK", "name": "腾讯控股", "market": "HK"},
    ]
    market = MagicMock(spec=MarketClient)
    ctx = ToolContext(
        session_id="s1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
        geegoo_bot_client=geegoo,
    )
    result = registry.execute(
        ToolCallRequest(name="search_code", arguments={"regex": "腾讯"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["count"] == 1
    geegoo.post_direct.assert_called_once()
    call_args = geegoo.post_direct.call_args
    assert call_args[0][0] == "/searchCode"
    assert call_args[0][1]["regex"] == "腾讯"


@pytest.mark.unit
def test_get_current_price_uses_direct_response(tmp_path) -> None:
    registry = register_all_tools(ToolRegistry())
    geegoo = MagicMock()
    geegoo.post_direct.return_value = {"price": 320.5}
    market = MagicMock(spec=MarketClient)
    ctx = ToolContext(
        session_id="s1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
        geegoo_bot_client=geegoo,
    )
    result = registry.execute(
        ToolCallRequest(name="get_current_price", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data["price"] == 320.5
    assert result.data["source"] == "5700"
    assert "320.5" in result.summary
    body = geegoo.post_direct.call_args[0][1]
    assert body == {"code": "00700.HK", "mcp_token": "mcp-test"}


@pytest.mark.unit
def test_get_current_price_falls_back_to_ticker(tmp_path) -> None:
    from geegoo_agent.exceptions import ClientError

    registry = register_all_tools(ToolRegistry())
    geegoo = MagicMock()
    geegoo.post_direct.side_effect = ClientError("server error 500 for /getCurrentPrice")
    market = MagicMock(spec=MarketClient)
    market.post.return_value = {
        "code": 100,
        "data": [{"price": 1688.0, "time": "15:00:00"}],
    }
    ctx = ToolContext(
        session_id="s1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
        geegoo_bot_client=geegoo,
    )
    result = registry.execute(
        ToolCallRequest(name="get_current_price", arguments={"code": "600519.SH"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data["price"] == 1688.0
    assert result.data["source"] == "5700/get_ticker"


@pytest.mark.unit
def test_http_api_tool_dry_run_skips_http(tmp_path) -> None:
    registry = register_all_tools(ToolRegistry())
    geegoo = MagicMock(spec=GeeGooBotClient)
    market = MagicMock(spec=MarketClient)
    ctx = ToolContext(
        session_id="s1",
        mcp_token="mcp-test",
        dry_run=True,
        workspace_root=tmp_path,
        market_client=market,
        geegoo_bot_client=geegoo,
    )
    result = registry.execute(
        ToolCallRequest(name="get_ticker", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "dry_run"
    market.post.assert_not_called()


@pytest.mark.unit
def test_scheduled_mode_hides_bot_mutations() -> None:
    registry = register_all_tools(ToolRegistry())
    names = {schema.name for schema in registry.schemas(mode="scheduled")}
    assert not names & BOT_MUTATION_TOOLS
    assert "check_trading_day" in names


@pytest.mark.unit
def test_bespoke_tools_keep_custom_category() -> None:
    registry = register_all_tools(ToolRegistry())
    tool = registry.get("check_trading_day")
    assert tool.category == ToolCategory.PERCEPTION
