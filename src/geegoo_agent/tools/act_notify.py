"""Notification action tools."""

from __future__ import annotations

import httpx
from pydantic import BaseModel, Field

from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class SendFeishuSummaryParams(BaseModel):
    title: str = "盘前报告摘要"
    summary: str = Field(description="Short summary text for Feishu")
    code: str = ""


class SendFeishuSummaryTool(BaseTool):
    name = "send_feishu_summary"
    description = "Send a short summary to Feishu webhook (optional)."
    category = ToolCategory.ACTION
    input_model = SendFeishuSummaryParams

    def run(self, params: SendFeishuSummaryParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary="dry-run: skipped Feishu push",
                data={"sent": False},
            )
        if not ctx.feishu_webhook_url:
            return ToolResult(
                status="skipped",
                summary="Feishu webhook not configured",
                data={"sent": False},
            )
        payload = {
            "msg_type": "text",
            "content": {"text": f"{params.title}\n{params.summary}"[:2000]},
        }
        response = httpx.post(ctx.feishu_webhook_url, json=payload, timeout=30.0)
        response.raise_for_status()
        return ToolResult(
            status="ok",
            summary="Feishu summary sent",
            data={"sent": True, "code": params.code},
        )
