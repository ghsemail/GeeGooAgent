"""Unit tests for cross-session recall."""

from __future__ import annotations

import pytest

from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.llm.types import Message, ToolCall
from geegoo_agent.runtime.chat_session import ChatSession, ChatSessionStore
from geegoo_agent.runtime.session_memory import extract_stock_events, search_past_sessions
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.bootstrap import register_all_tools
from geegoo_agent.tools.types import ToolCallRequest, ToolContext


@pytest.mark.unit
def test_extract_stock_events_from_tool_results() -> None:
    session = ChatSession(id="chat-old")
    session.append_message(Message(role="user", content="查腾讯股价"))
    session.append_message(
        Message(
            role="assistant",
            content=None,
            tool_calls=[
                ToolCall(id="c1", name="get_current_price", arguments={"code": "00700.HK"})
            ],
        )
    )
    session.append_message(
        Message(
            role="tool",
            content='{"status":"ok","summary":"00700.HK price=453.2","data":{"code":"00700.HK","price":453.2}}',
            tool_call_id="c1",
        )
    )
    events = extract_stock_events(session)
    assert len(events) == 1
    assert events[0].code == "00700.HK"
    assert events[0].price == 453.2


@pytest.mark.unit
def test_search_past_sessions_finds_closed_session(tmp_path) -> None:
    store = ChatSessionStore(FileStateStore(tmp_path))
    old = ChatSession(id="chat-old1", status="closed")
    old.append_message(Message(role="system", content="sys"))
    old.append_message(Message(role="user", content="查一下腾讯控股股价"))
    old.append_message(
        Message(
            role="assistant",
            content=None,
            tool_calls=[
                ToolCall(id="c1", name="get_current_price", arguments={"code": "00700.HK"})
            ],
        )
    )
    old.append_message(
        Message(
            role="tool",
            content='{"status":"ok","summary":"00700.HK price=453.2","data":{"code":"00700.HK","price":453.2}}',
            tool_call_id="c1",
        )
    )
    store.save(old)

    current = ChatSession(id="chat-new1")
    store.save(current)

    hits = search_past_sessions(
        store,
        "腾讯 股价",
        exclude_session_id=current.id,
        limit=3,
    )
    assert hits
    assert hits[0].session_id == "chat-old1"
    assert any(e.code == "00700.HK" for e in hits[0].stock_events)


@pytest.mark.unit
def test_recall_tool_via_registry(tmp_path) -> None:
    store = ChatSessionStore(FileStateStore(tmp_path))
    old = ChatSession(id="chat-old2", status="closed")
    old.append_message(Message(role="user", content="茅台价格"))
    old.append_message(
        Message(
            role="assistant",
            content=None,
            tool_calls=[
                ToolCall(id="c1", name="get_current_price", arguments={"code": "600519.SH"})
            ],
        )
    )
    old.append_message(
        Message(
            role="tool",
            content='{"status":"ok","summary":"600519.SH price=1688","data":{"code":"600519.SH","price":1688}}',
            tool_call_id="c1",
        )
    )
    store.save(old)

    registry = register_all_tools(ToolRegistry())
    ctx = ToolContext(
        session_id="chat-current",
        mcp_token="mcp",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=__import__("unittest.mock", fromlist=["MagicMock"]).MagicMock(),
        state_store=store._store,
    )
    result = registry.execute(
        ToolCallRequest(name="recall", arguments={"query": "茅台 价格"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data is not None
    assert result.data["count"] >= 1
    assert "600519.SH" in result.summary
