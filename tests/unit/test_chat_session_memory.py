"""Chat session memory helpers."""

from __future__ import annotations

import pytest

from geegoo_agent.runtime.chat_session import ChatSession
from geegoo_agent.llm.types import Message, ToolCall


@pytest.mark.unit
def test_tool_activity_summary_lists_prior_calls() -> None:
    session = ChatSession(id="chat-mem")
    session.append_message(Message(role="system", content="sys"))
    session.append_message(
        Message(
            role="assistant",
            content=None,
            tool_calls=[ToolCall(id="c1", name="search_code", arguments={"regex": "茅台"})],
        )
    )
    session.append_message(
        Message(
            role="assistant",
            content=None,
            tool_calls=[
                ToolCall(id="c2", name="get_current_price", arguments={"code": "600519.SH"})
            ],
        )
    )
    summary = session.tool_activity_summary()
    assert "search_code" in summary
    assert "get_current_price" in summary
    assert "600519.SH" in summary
