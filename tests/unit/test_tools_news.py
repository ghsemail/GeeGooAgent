"""Unit tests for news tools."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from geegoo_agent.clients.market import MarketClient
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext

PROJECT_ROOT = Path(__file__).resolve().parents[2]


@pytest.fixture
def registry() -> ToolRegistry:
    return register_mvp_tools(ToolRegistry())


@pytest.fixture
def ctx(tmp_path) -> ToolContext:
    return ToolContext(
        session_id="sess-1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=MagicMock(spec=MarketClient),
        project_root=PROJECT_ROOT,
    )


@pytest.mark.unit
@patch("geegoo_agent.tools.news._run_script", return_value="news line 1")
def test_fetch_market_news_us(mock_run, registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(name="fetch_market_news", arguments={"market": "US", "limit": 3}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data["text"] == "news line 1"
    mock_run.assert_called_once()


@pytest.mark.unit
@patch("geegoo_agent.tools.news._run_script", return_value="stock news")
def test_fetch_stock_news(mock_run, registry: ToolRegistry, ctx: ToolContext) -> None:
    result = registry.execute(
        ToolCallRequest(
            name="fetch_stock_news",
            arguments={"code": "00700.HK", "stock_name": "腾讯控股"},
        ),
        ctx,
    )
    assert result.status == "ok"
    assert result.data["source"] == "eastmoney"


@pytest.mark.unit
def test_fetch_market_news_dry_run(registry: ToolRegistry, ctx: ToolContext) -> None:
    ctx.dry_run = True
    result = registry.execute(
        ToolCallRequest(name="fetch_market_news", arguments={"market": "HK"}),
        ctx,
    )
    assert result.status == "dry_run"
