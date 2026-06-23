"""Analysis tools — MCP analysis, capital, attitude."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

from geegoo_agent.clients.market import CapitalFlowPeriod
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.mappings import (
    A_SHARE_CAPITAL_SKIP_REASON,
    attitude_to_result,
    format_capital_distribution,
    is_a_share,
)
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult

INDEX_PROMPT_ID = "69ec7035b9ccd3d9befc6c23"
McpPeriod = Literal[
    "no_period",
    "minutes",
    "hourly",
    "daily",
    "weekly",
    "monthly",
    "quarterly",
    "yearly",
    "longterm",
]


class GetMcpAnalysisParams(BaseModel):
    name: str = Field(description="Stock or index display name")
    code: str = Field(description="Ticker code")
    prompt_id: str = Field(default=INDEX_PROMPT_ID)
    period: McpPeriod = Field(description="Analysis period, e.g. hourly or weekly")
    language: str = "cn"


class GetCapitalFlowParams(BaseModel):
    code: str
    period: CapitalFlowPeriod = "DAY"


class GetCapitalDistributionParams(BaseModel):
    code: str


class GetBotYesterdayAttitudeParams(BaseModel):
    bot_id: str
    code: str = ""
    language: str = "cn"


class GetMcpAnalysisTool(BaseTool):
    name = "get_mcp_analysis"
    description = "Run MCP technical analysis for an index or stock."
    category = ToolCategory.ANALYSIS
    input_model = GetMcpAnalysisParams

    def run(self, params: GetMcpAnalysisParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped get_mcp_analysis {params.code}",
                data={
                    "code": params.code,
                    "period": params.period,
                    "analysis_result": f"[dry-run] hourly analysis for {params.name}",
                },
            )
        result = ctx.market_client.get_mcp_analysis(
            ctx.mcp_token,
            name=params.name,
            code=params.code,
            prompt_id=params.prompt_id,
            period=params.period,
            language=params.language,
        )
        return ToolResult(
            status="ok",
            summary=f"MCP analysis for {params.code} ({params.period})",
            data={
                "code": params.code,
                "period": params.period,
                "analysis_result": result.analysis_result,
                "model": result.model,
                "create_date": result.create_date,
            },
        )


class GetCapitalFlowTool(BaseTool):
    name = "get_capital_flow"
    description = "Fetch capital flow for a stock (period=DAY for pre-market)."
    category = ToolCategory.ANALYSIS
    input_model = GetCapitalFlowParams

    def run(self, params: GetCapitalFlowParams, ctx: ToolContext) -> ToolResult:
        if is_a_share(params.code):
            return ToolResult(
                status="skipped",
                summary=f"skipped get_capital_flow {params.code}: {A_SHARE_CAPITAL_SKIP_REASON}",
                data={"code": params.code, "items": [], "latest": {}, "skip_reason": A_SHARE_CAPITAL_SKIP_REASON},
            )
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped get_capital_flow {params.code}",
                data={"code": params.code, "items": []},
            )
        items = ctx.market_client.get_capital_flow(
            ctx.mcp_token,
            params.code,
            period=params.period,
        )
        latest = items[-1].model_dump() if items else {}
        return ToolResult(
            status="ok",
            summary=f"Capital flow for {params.code}: {len(items)} item(s)",
            data={"code": params.code, "items": [i.model_dump() for i in items], "latest": latest},
        )


class GetCapitalDistributionTool(BaseTool):
    name = "get_capital_distribution"
    description = "Fetch T-1 capital distribution for a stock."
    category = ToolCategory.ANALYSIS
    input_model = GetCapitalDistributionParams

    def run(self, params: GetCapitalDistributionParams, ctx: ToolContext) -> ToolResult:
        if is_a_share(params.code):
            return ToolResult(
                status="skipped",
                summary=f"skipped get_capital_distribution {params.code}: {A_SHARE_CAPITAL_SKIP_REASON}",
                data={
                    "code": params.code,
                    "formatted": A_SHARE_CAPITAL_SKIP_REASON,
                    "skip_reason": A_SHARE_CAPITAL_SKIP_REASON,
                },
            )
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped get_capital_distribution {params.code}",
                data={"code": params.code, "formatted": ""},
            )
        dist = ctx.market_client.get_capital_distribution(ctx.mcp_token, params.code)
        payload = dist.model_dump()
        return ToolResult(
            status="ok",
            summary=f"Capital distribution for {params.code}",
            data={
                "code": params.code,
                "raw": payload,
                "formatted": format_capital_distribution(payload),
            },
        )


class GetBotYesterdayAttitudeTool(BaseTool):
    name = "get_bot_yesterday_attitude"
    description = "获取上一交易日 Bot 态度（服务端自动回溯最近 7 天）；无记录兜底为 neutral。"
    category = ToolCategory.ANALYSIS
    input_model = GetBotYesterdayAttitudeParams

    def run(self, params: GetBotYesterdayAttitudeParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: neutral attitude for {params.bot_id}",
                data={
                    "bot_id": params.bot_id,
                    "code": params.code,
                    "attitude": "neutral",
                    "result": "neutral",
                    "found": False,
                },
            )
        attitude = ctx.market_client.get_bot_yesterday_attitude(
            ctx.mcp_token,
            params.bot_id,
            language=params.language,
        )
        mapped = attitude_to_result(attitude.attitude)
        date_info = f" @ {attitude.date}" if attitude.date else ""
        return ToolResult(
            status="ok",
            summary=(
                f"Attitude for {params.bot_id}{date_info}: {attitude.attitude}"
                if attitude.found
                else f"无上一交易日 attitude 记录 {params.bot_id}，兜底 neutral"
            ),
            data={
                **attitude.model_dump(),
                "code": params.code or attitude.code,
                "result": mapped,
            },
        )
