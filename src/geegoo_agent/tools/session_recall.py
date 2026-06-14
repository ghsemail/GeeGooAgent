"""Recall past chat sessions (Hermes-style session memory)."""

from __future__ import annotations

from pydantic import BaseModel, Field

from geegoo_agent.exceptions import ConfigError
from geegoo_agent.runtime.chat_session import ChatSessionStore
from geegoo_agent.runtime.session_memory import hits_to_data, search_past_sessions
from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult


class RecallParams(BaseModel):
    query: str = Field(
        default="",
        description='Keywords to search past chat sessions, e.g. "腾讯 股价" or "股票 价格"',
    )
    limit: int = Field(default=5, ge=1, le=20, description="Max sessions to return")


class RecallSessionTool(BaseTool):
    name = "recall"
    description = (
        "Search past geegoo chat sessions for stock price lookups and user queries. "
        "Use when the user asks what they checked before, including after quit/restart."
    )
    category = ToolCategory.META
    input_model = RecallParams

    def run(self, params: RecallParams, ctx: ToolContext) -> ToolResult:
        if ctx.state_store is None:
            raise ConfigError("state_store not configured")
        store = ChatSessionStore(ctx.state_store)
        hits = search_past_sessions(
            store,
            params.query,
            exclude_session_id=ctx.session_id,
            limit=params.limit,
        )
        if not hits:
            return ToolResult(
                status="ok",
                summary="No matching past chat sessions",
                data={"count": 0, "matches": []},
            )
        top = hits[0]
        priced = [e for e in top.stock_events if e.code and e.tool == "get_current_price"]
        if priced:
            last = priced[-1]
            price_note = f" @ {last.price}" if last.price is not None else ""
            summary = f"Found {len(hits)} session(s); latest: {last.code}{price_note} ({top.session_id})"
        else:
            summary = f"Found {len(hits)} session(s); latest: {top.snippet} ({top.session_id})"
        return ToolResult(status="ok", summary=summary, data=hits_to_data(hits))
