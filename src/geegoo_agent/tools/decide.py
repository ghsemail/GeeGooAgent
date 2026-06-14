"""Decision / memory read tools."""

from __future__ import annotations

from datetime import date, timedelta

from pydantic import BaseModel

from geegoo_agent.exceptions import ConfigError
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class RecallYesterdaySummaryParams(BaseModel):
    code: str


class ReadWorkingStateParams(BaseModel):
    pass


class RecallYesterdaySummaryTool(BaseTool):
    name = "recall_yesterday_summary"
    description = "Recall yesterday's local pre-market report summary for a code."
    category = ToolCategory.DECISION
    input_model = RecallYesterdaySummaryParams

    def run(self, params: RecallYesterdaySummaryParams, ctx: ToolContext) -> ToolResult:
        yesterday = (date.today() - timedelta(days=1)).isoformat()
        path = ctx.workspace_root / "reports" / yesterday / f"{params.code}-premarket.md"
        if not path.exists():
            return ToolResult(
                status="ok",
                summary=f"No yesterday report for {params.code}",
                data={"code": params.code, "found": False, "summary": ""},
            )
        text = path.read_text(encoding="utf-8")
        summary = _extract_summary(text)
        return ToolResult(
            status="ok",
            summary=f"Recalled yesterday report for {params.code}",
            data={"code": params.code, "found": True, "summary": summary, "path": str(path)},
        )


class ReadWorkingStateTool(BaseTool):
    name = "read_working_state"
    description = "Read structured working memory for the current session."
    category = ToolCategory.META
    input_model = ReadWorkingStateParams

    def run(self, params: ReadWorkingStateParams, ctx: ToolContext) -> ToolResult:
        if ctx.working_store is None:
            raise ConfigError("working_store not configured")
        working = ctx.working_store.load(ctx.session_id)
        if working is None:
            return ToolResult(
                status="error",
                summary=f"working state not found for session {ctx.session_id}",
            )
        summary = ctx.working_store.summary(working)
        return ToolResult(
            status="ok",
            summary=summary,
            data=working.model_dump(),
        )


def _extract_summary(text: str, max_len: int = 500) -> str:
    lines = [line.strip() for line in text.splitlines() if line.strip()]
    body = "\n".join(lines[:20])
    if len(body) > max_len:
        return body[:max_len] + "…"
    return body
