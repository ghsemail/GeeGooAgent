"""Report action tools — create, save, query."""

from __future__ import annotations

from datetime import date
from typing import Literal

from pydantic import BaseModel, Field

from geegoo_agent.exceptions import ConfigError
from geegoo_agent.infra.sandbox import WorkspaceGuard
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.schemas import PreMarketReportCreate
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class CreatePreMarketReportParams(PreMarketReportCreate):
    pass


class SaveLocalReportParams(BaseModel):
    code: str
    content: str
    report_type: Literal["premarket", "postmarket", "intraday"] = "premarket"
    report_date: str | None = Field(
        default=None,
        description="YYYY-MM-DD; defaults to today",
    )


class GetStockDailyReportsParams(BaseModel):
    code: str
    report_date: str = Field(description="YYYY-MM-DD")


class ListTodayReportsParams(BaseModel):
    code: str
    report_date: str | None = None


class CreatePreMarketReportTool(BaseTool):
    name = "create_pre_market_report"
    description = "Create a pre-market report via API (validates required fields)."
    category = ToolCategory.ACTION
    input_model = CreatePreMarketReportParams

    def run(self, params: CreatePreMarketReportParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped create_pre_market_report {params.code}",
                data={"report_id": "dry-run-id", "code": params.code},
            )
        body = params.to_api_body()
        result = ctx.market_client.create_pre_market_report(ctx.mcp_token, body)
        return ToolResult(
            status="ok",
            summary=f"Created pre-market report for {params.code}",
            data={"report_id": result.report_id, "code": params.code},
        )


class SaveLocalReportTool(BaseTool):
    name = "save_local_report"
    description = "Save report markdown under workspace reports/."
    category = ToolCategory.ACTION
    input_model = SaveLocalReportParams

    def run(self, params: SaveLocalReportParams, ctx: ToolContext) -> ToolResult:
        report_date = params.report_date or date.today().isoformat()
        guard = WorkspaceGuard(ctx.workspace_root)
        suffix = {
            "premarket": "premarket",
            "postmarket": "postmarket",
            "intraday": "intraday",
        }[params.report_type]
        rel = f"reports/{report_date}/{params.code}-{suffix}.md"
        path = guard.resolve(rel)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(params.content, encoding="utf-8")
        return ToolResult(
            status="ok",
            summary=f"Saved local report {path.name}",
            data={"path": str(path), "code": params.code},
        )


class GetStockDailyReportsTool(BaseTool):
    name = "get_stock_daily_reports"
    description = "Query aggregated daily reports from geegoo (5700)."
    category = ToolCategory.ANALYSIS
    input_model = GetStockDailyReportsParams

    def run(self, params: GetStockDailyReportsParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: empty daily reports for {params.code}",
                data={"pre_market": [], "intraday": [], "post_market": []},
            )
        if ctx.geegoo_bot_client is None:
            raise ConfigError("geegoo_bot_client not configured")
        reports = ctx.geegoo_bot_client.get_stock_daily_reports(
            ctx.mcp_token,
            params.code,
            params.report_date,
        )
        return ToolResult(
            status="ok",
            summary=(
                f"Daily reports {params.code} @ {params.report_date}: "
                f"pre={len(reports.pre_market)}"
            ),
            data=reports.model_dump(),
        )


class ListTodayReportsTool(BaseTool):
    name = "list_today_reports"
    description = "Idempotency check — list today's pre_market reports for a code."
    category = ToolCategory.ANALYSIS
    input_model = ListTodayReportsParams

    def run(self, params: ListTodayReportsParams, ctx: ToolContext) -> ToolResult:
        report_date = params.report_date or date.today().isoformat()
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: no existing reports for {params.code}",
                data={"code": params.code, "report_date": report_date, "count": 0, "reports": []},
            )
        if ctx.geegoo_bot_client is None:
            raise ConfigError("geegoo_bot_client not configured")
        reports = ctx.geegoo_bot_client.get_stock_daily_reports(
            ctx.mcp_token,
            params.code,
            report_date,
        )
        pre = reports.pre_market
        return ToolResult(
            status="ok",
            summary=f"Found {len(pre)} pre_market report(s) for {params.code} on {report_date}",
            data={
                "code": params.code,
                "report_date": report_date,
                "count": len(pre),
                "reports": pre,
                "already_reported": len(pre) > 0,
            },
        )
