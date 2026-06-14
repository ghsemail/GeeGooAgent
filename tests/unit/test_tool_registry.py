"""Unit tests for ToolRegistry."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock

import pytest

from geegoo_agent.clients.market import MarketClient, TradingDayData, UserBotCode
from geegoo_agent.infra.events import InProcessEventBus
from geegoo_agent.tools.bootstrap import register_all_tools, register_mvp_tools
from geegoo_agent.tools.registry import ToolNotFoundError, ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext

PROJECT_ROOT = Path(__file__).resolve().parents[2]


@pytest.fixture
def registry() -> ToolRegistry:
    return register_mvp_tools(ToolRegistry(), project_root=PROJECT_ROOT)


@pytest.fixture
def full_registry() -> ToolRegistry:
    return register_all_tools(ToolRegistry())


@pytest.fixture
def tool_ctx(tmp_path) -> ToolContext:
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
            bot_name="DCA",
            bot_type="DCA",
        )
    ]
    return ToolContext(
        session_id="sess-1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=market,
        event_bus=InProcessEventBus(),
    )


@pytest.mark.unit
def test_register_mvp_tools_count(registry: ToolRegistry) -> None:
    names = registry.list_names()
    assert len(names) == 16
    assert "check_trading_day" in names
    assert "create_pre_market_report" in names
    assert "fetch_market_news" in names
    assert "list_today_reports" in names


@pytest.mark.unit
def test_register_all_tools_includes_catalog(full_registry: ToolRegistry) -> None:
    names = full_registry.list_names()
    assert "search_code" in names
    assert "create_dca_bot" in names
    assert "loopback_strategy" in names
    assert len(names) >= 70


@pytest.mark.unit
def test_unknown_tool_raises(registry: ToolRegistry, tool_ctx: ToolContext) -> None:
    with pytest.raises(ToolNotFoundError):
        registry.execute(ToolCallRequest(name="missing_tool"), tool_ctx)


@pytest.mark.unit
def test_schemas_respects_tool_filter(registry: ToolRegistry) -> None:
    schemas = registry.schemas(tool_filter=["check_trading_day"])
    assert len(schemas) == 1
    assert schemas[0].name == "check_trading_day"


@pytest.mark.unit
def test_execute_emits_events(registry: ToolRegistry, tool_ctx: ToolContext) -> None:
    bus = tool_ctx.event_bus
    assert bus is not None
    registry.execute(
        ToolCallRequest(name="check_trading_day", arguments={"code": "00700.HK"}),
        tool_ctx,
    )
    events = [name for name, _ in bus.history]
    assert "ToolCalled" in events
    assert "ToolCompleted" in events


@pytest.mark.unit
def test_scheduled_mode_mvp_has_no_bot_mutation_tools(registry: ToolRegistry) -> None:
    names = [s.name for s in registry.schemas(mode="scheduled")]
    assert "create_dca_bot" not in names
    assert "check_trading_day" in names


@pytest.mark.unit
def test_scheduled_mode_full_registry_hides_mutations(full_registry: ToolRegistry) -> None:
    names = [s.name for s in full_registry.schemas(mode="scheduled")]
    assert "create_dca_bot" not in names
    assert "switch_bot" not in names
