"""Register tools on a registry — bespoke + HTTP catalog."""

from __future__ import annotations

from pathlib import Path

from geegoo_agent.runtime.skill_loader import SkillLoader
from geegoo_agent.tools.act_notify import SendFeishuSummaryTool
from geegoo_agent.tools.act_reports import (
    CreatePreMarketReportTool,
    GetStockDailyReportsTool,
    ListTodayReportsTool,
    SaveLocalReportTool,
)
from geegoo_agent.tools.analyze import (
    GetBotYesterdayAttitudeTool,
    GetCapitalDistributionTool,
    GetCapitalFlowTool,
    GetMcpAnalysisTool,
)
from geegoo_agent.tools.catalog import catalog_http_specs
from geegoo_agent.tools.decide import ReadWorkingStateTool, RecallYesterdaySummaryTool
from geegoo_agent.tools.http_api import build_http_tools
from geegoo_agent.tools.meta import WriteExecutionLogTool
from geegoo_agent.tools.session_recall import RecallSessionTool
from geegoo_agent.tools.news import FetchMarketNewsTool, FetchStockNewsTool
from geegoo_agent.tools.perceive import (
    CheckTradingDayTool,
    GetCurrentPriceTool,
    GetReportBotCodesTool,
)
from geegoo_agent.tools.registry import ToolRegistry


def bespoke_tool_instances():
    return [
        CheckTradingDayTool(),
        GetCurrentPriceTool(),
        GetReportBotCodesTool(),
        FetchMarketNewsTool(),
        FetchStockNewsTool(),
        GetMcpAnalysisTool(),
        GetStockDailyReportsTool(),
        ListTodayReportsTool(),
        GetCapitalFlowTool(),
        GetCapitalDistributionTool(),
        GetBotYesterdayAttitudeTool(),
        RecallYesterdaySummaryTool(),
        ReadWorkingStateTool(),
        CreatePreMarketReportTool(),
        SaveLocalReportTool(),
        SendFeishuSummaryTool(),
        WriteExecutionLogTool(),
        RecallSessionTool(),
    ]


def all_mvp_tool_instances():
    """Backward-compatible alias — returns full catalog."""
    return all_tool_instances()


def all_tool_instances():
    bespoke = {tool.name: tool for tool in bespoke_tool_instances()}
    http = {tool.name: tool for tool in build_http_tools(catalog_http_specs())}
    merged = {**http, **bespoke}
    return list(merged.values())


def register_all_tools(
    registry: ToolRegistry,
    *,
    tool_filter: list[str] | None = None,
) -> ToolRegistry:
    by_name = {tool.name: tool for tool in all_tool_instances()}
    names = tool_filter if tool_filter is not None else sorted(by_name.keys())
    for name in names:
        registry.register(by_name[name])
    return registry


def register_mvp_tools(
    registry: ToolRegistry,
    *,
    tool_filter: list[str] | None = None,
    project_root: Path | None = None,
) -> ToolRegistry:
    """Register tools; defaults to pre_market manifest whitelist when project_root set."""
    if tool_filter is None and project_root is not None:
        try:
            manifest = SkillLoader(project_root).load("pre_market")
            tool_filter = manifest.tools
        except Exception:
            tool_filter = None
    if tool_filter is None:
        return register_all_tools(registry)
    return register_all_tools(registry, tool_filter=tool_filter)
