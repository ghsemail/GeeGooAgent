"""Per-tool dry-run and mocked execution tests for all 82 registered tools."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest
from tests.helpers.tool_samples import is_bespoke_tool, is_http_tool, sample_arguments

from geegoo_agent.clients.geegoo_bot import DailyReportsData
from geegoo_agent.clients.market import (
    BotYesterdayAttitude,
    CapitalDistributionData,
    CapitalFlowItem,
    McpAnalysisResult,
    PreMarketReportResult,
    TradingDayData,
    UserBotCode,
)
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.tools.bootstrap import all_tool_instances, register_all_tools
from geegoo_agent.tools.catalog import SPEC_BY_NAME
from geegoo_agent.tools.http_api import HttpApiTool
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext

ALL_TOOLS = sorted(all_tool_instances(), key=lambda t: t.name)
ALL_TOOL_NAMES = [t.name for t in ALL_TOOLS]
HTTP_TOOL_NAMES = [t.name for t in ALL_TOOLS if is_http_tool(t)]
BESPOKE_TOOL_NAMES_LIST = [t.name for t in ALL_TOOLS if is_bespoke_tool(t)]


@pytest.fixture
def registry() -> ToolRegistry:
    return register_all_tools(ToolRegistry())


def _mock_market() -> MagicMock:
    market = MagicMock()
    market.post.return_value = {"code": 100, "data": {"ok": True}}
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
    market.get_mcp_analysis.return_value = McpAnalysisResult(
        analysis_result="hourly analysis",
        model="test",
        create_date="2026-06-05",
    )
    market.get_capital_flow.return_value = [
        CapitalFlowItem(in_flow=1.0, capital_flow_item_time="2026-06-05"),
    ]
    market.get_capital_distribution.return_value = CapitalDistributionData(
        capital_in_super=1e8,
        capital_out_super=0.5e8,
    )
    market.get_bot_yesterday_attitude.return_value = BotYesterdayAttitude(
        attitude="bullish",
        bot_id="bot-1",
        code="00700.HK",
        found=True,
    )
    market.create_pre_market_report.return_value = PreMarketReportResult(report_id="rpt-1")
    return market


def _mock_geegoo() -> MagicMock:
    geegoo = MagicMock()
    geegoo.post.return_value = {"code": 100, "data": {"items": []}}
    geegoo.post_direct.return_value = [{"code": "00700.HK", "name": "腾讯控股"}]
    geegoo.get_stock_daily_reports.return_value = DailyReportsData(
        pre_market=[],
        intraday=[],
        post_market=[],
    )
    return geegoo


def _tool_context(
    tmp_path,
    *,
    dry_run: bool,
    market: MagicMock | None = None,
    geegoo: MagicMock | None = None,
    working_store: WorkingMemoryStore | None = None,
    feishu_webhook_url: str | None = None,
) -> ToolContext:
    return ToolContext(
        session_id="sess-all-tools",
        mcp_token="mcp-test-token",
        dry_run=dry_run,
        workspace_root=tmp_path,
        market_client=market or _mock_market(),
        geegoo_bot_client=geegoo or _mock_geegoo(),
        working_store=working_store,
        feishu_webhook_url=feishu_webhook_url,
    )


@pytest.mark.unit
@pytest.mark.parametrize("tool_name", ALL_TOOL_NAMES)
def test_tool_dry_run(registry: ToolRegistry, tool_name: str, tmp_path) -> None:
    tool = registry.get(tool_name)
    args = sample_arguments(tool)
    ctx = _tool_context(tmp_path, dry_run=True)
    if tool_name == "read_working_state":
        store = FileStateStore(tmp_path)
        ws = WorkingMemoryStore(store)
        ws.create("sess-all-tools", skill="pre_market")
        ctx.working_store = ws
    if tool_name == "recall":
        ctx.state_store = FileStateStore(tmp_path)
    result = registry.execute(ToolCallRequest(name=tool_name, arguments=args), ctx)
    assert result.status in {"ok", "dry_run", "skipped"}, (
        f"{tool_name} dry_run failed: {result.summary}"
    )


@pytest.mark.unit
@pytest.mark.parametrize("tool_name", HTTP_TOOL_NAMES)
def test_http_tool_executes_with_mock(registry: ToolRegistry, tool_name: str, tmp_path) -> None:
    tool = registry.get(tool_name)
    assert isinstance(tool, HttpApiTool)
    spec = SPEC_BY_NAME[tool_name]
    args = sample_arguments(tool)
    market = _mock_market()
    geegoo = _mock_geegoo()
    ctx = _tool_context(tmp_path, dry_run=False, market=market, geegoo=geegoo)

    result = registry.execute(ToolCallRequest(name=tool_name, arguments=args), ctx)
    assert result.status == "ok", f"{tool_name}: {result.summary}"

    client = market if spec.client == "market" else geegoo
    if spec.response_mode == "direct":
        client.post_direct.assert_called_once()
        call_path, call_body = client.post_direct.call_args[0]
    else:
        client.post.assert_called_once()
        call_path, call_body = client.post.call_args[0]

    assert call_path == spec.path
    if spec.requires_mcp_token:
        assert call_body.get("mcp_token") == "mcp-test-token"
    else:
        assert "mcp_token" not in call_body


@pytest.mark.unit
@pytest.mark.parametrize("tool_name", BESPOKE_TOOL_NAMES_LIST)
def test_bespoke_tool_executes_with_mock(
    registry: ToolRegistry, tool_name: str, tmp_path
) -> None:
    tool = registry.get(tool_name)
    args = sample_arguments(tool)
    market = _mock_market()
    geegoo = _mock_geegoo()
    store = FileStateStore(tmp_path)
    working_store = WorkingMemoryStore(store)
    working_store.create("sess-all-tools", skill="pre_market")
    ctx = _tool_context(
        tmp_path,
        dry_run=False,
        market=market,
        geegoo=geegoo,
        working_store=working_store,
    )
    ctx.state_store = store

    if tool_name in {"fetch_market_news", "fetch_stock_news"}:
        with patch("geegoo_agent.tools.news._run_script", return_value="news text"):
            result = registry.execute(ToolCallRequest(name=tool_name, arguments=args), ctx)
    elif tool_name == "send_feishu_summary":
        with patch("httpx.post") as mock_post:
            mock_resp = MagicMock()
            mock_resp.raise_for_status.return_value = None
            mock_post.return_value = mock_resp
            ctx.feishu_webhook_url = "https://feishu.test/hook"
            result = registry.execute(ToolCallRequest(name=tool_name, arguments=args), ctx)
    elif tool_name == "get_current_price":
        geegoo.post_direct.return_value = {"price": 350.0}
        result = registry.execute(ToolCallRequest(name=tool_name, arguments=args), ctx)
    else:
        result = registry.execute(ToolCallRequest(name=tool_name, arguments=args), ctx)

    assert result.status in {"ok", "skipped"}, f"{tool_name}: {result.summary}"

    if tool_name == "check_trading_day":
        market.check_trading_day.assert_called_once()
    elif tool_name == "get_report_bot_codes":
        market.get_report_bot_codes.assert_called_once()
    elif tool_name == "get_mcp_analysis":
        market.get_mcp_analysis.assert_called_once()
    elif tool_name == "get_capital_flow":
        market.get_capital_flow.assert_called_once()
    elif tool_name == "get_capital_distribution":
        market.get_capital_distribution.assert_called_once()
    elif tool_name == "get_bot_yesterday_attitude":
        market.get_bot_yesterday_attitude.assert_called_once()
    elif tool_name in {"get_stock_daily_reports", "list_today_reports"}:
        geegoo.get_stock_daily_reports.assert_called_once()
    elif tool_name == "create_pre_market_report":
        market.create_pre_market_report.assert_called_once()
    elif tool_name == "save_local_report":
        assert result.data is not None
        assert "path" in result.data
    elif tool_name == "write_execution_log":
        assert result.data is not None
        assert "path" in result.data
    elif tool_name == "read_working_state":
        assert result.data is not None
    elif tool_name == "send_feishu_summary":
        assert result.data is not None
        assert result.data.get("sent") is True


@pytest.mark.unit
def test_all_tools_count_matches_expectation() -> None:
    assert len(ALL_TOOL_NAMES) == 82
    assert len(HTTP_TOOL_NAMES) == 64
    assert len(BESPOKE_TOOL_NAMES_LIST) == 18
