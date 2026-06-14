"""Unit tests for live chat progress output."""

from __future__ import annotations

import io
from unittest.mock import MagicMock

import pytest

from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
from geegoo_agent.llm.types import LLMResponse, TokenUsage, ToolCall, ToolSchema
from geegoo_agent.runtime.chat_progress import make_progress_writer
from geegoo_agent.runtime.chat_ui import ChatUI
from geegoo_agent.runtime.chat_repl import ChatRepl
from geegoo_agent.tools.bootstrap import register_all_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.runtime.chat_tools import ON_DEMAND_CHAT_TOOLS
from geegoo_agent.runtime.chat_session import ChatSession, ChatSessionStore
from geegoo_agent.runtime.react_loop import ReActLoop
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.tools.types import ToolResult


class FakeProvider:
    def __init__(self, responses: list[LLMResponse]) -> None:
        self._responses = list(responses)
        self.model = "gpt-test"

    def chat(self, messages, tools, *, temperature=0.2, max_tokens=4096):
        return self._responses.pop(0)


@pytest.mark.unit
def test_progress_writer_shows_tool_flow() -> None:
    buffer = io.StringIO()
    emit = make_progress_writer(buffer)

    emit("round_start", {"round": 1, "step": 1})
    emit(
        "llm_plan",
        {
            "reasoning": "先搜索腾讯代码",
            "content": "",
            "tool_names": ["search_code"],
        },
    )
    emit("tool_start", {"name": "search_code", "arguments": {"regex": "腾讯"}})
    emit("tool_done", {"name": "search_code", "status": "ok", "summary": "1 item"})
    emit("reply_start", {"step": 2})

    out = buffer.getvalue()
    assert "[思考]" in out
    assert "search_code" in out
    assert "✓" in out
    assert "search_code" in out


@pytest.mark.unit
def test_chat_ui_rich_banner_does_not_crash() -> None:
    buffer = io.StringIO()
    ui = ChatUI(buffer, plain=False)
    registry = register_all_tools(ToolRegistry(), tool_filter=ON_DEMAND_CHAT_TOOLS)
    ui.print_banner(
        session_id="chat-1",
        provider="DeepSeek",
        model="deepseek-v4-flash",
        registry=registry,
        thinking=True,
        api_hosts={"market": "118.195.135.97:5700"},
    )
    out = buffer.getvalue()
    assert "GeeGoo Agent" in out
    assert "Available Tools" in out
    assert "perceive" in out or "analyze" in out


@pytest.mark.unit
def test_react_loop_emits_progress_events() -> None:
    events: list[str] = []

    def capture(event: str, data: dict) -> None:
        events.append(event)

    provider = FakeProvider(
        [
            LLMResponse(
                content="done",
                tool_calls=[],
                usage=TokenUsage(1, 1, "gpt-test"),
            ),
        ]
    )
    gateway = ModelGateway(provider, CostManager(), GatewayConfig(max_retries=1))
    loop = ReActLoop(gateway, MagicMock(), on_progress=capture)
    session = ChatSession(id="chat-events")
    loop.run_turn(session, "hi", MagicMock(), [])

    assert "round_start" in events
    assert "reply_start" in events


@pytest.mark.unit
def test_chat_repl_verbose_shows_live_steps(tmp_path, sample_config) -> None:
    provider = FakeProvider(
        [
            LLMResponse(
                content=None,
                tool_calls=[ToolCall(id="c1", name="search_code", arguments={"regex": "腾讯"})],
                usage=TokenUsage(10, 5, "gpt-test"),
            ),
            LLMResponse(
                content="腾讯 00700.HK",
                tool_calls=[],
                usage=TokenUsage(8, 12, "gpt-test"),
            ),
        ]
    )
    gateway = ModelGateway(provider, CostManager(), GatewayConfig(max_retries=1))
    executor = MagicMock()
    executor.execute.return_value = ToolResult(status="ok", summary="search_code: 1 item(s)")

    store = FileStateStore(tmp_path)
    session = ChatSession(id="chat-verbose")
    session_store = ChatSessionStore(store)

    app = MagicMock()
    app.state_store = store
    app.config = sample_config
    app.llm_gateway = gateway
    app.executor = executor
    app.event_bus.history = []
    app._tool_context.return_value = MagicMock(dry_run=False)

    stdout = io.StringIO()
    repl = ChatRepl(
        app=app,
        session=session,
        session_store=session_store,
        registry=MagicMock(list_names=lambda: [], schemas=lambda **_: []),
        loop=ReActLoop(gateway, executor),
        stdout=stdout,
        verbose=True,
    )
    repl._attach_progress()

    repl._handle_user_message("查腾讯")
    out = stdout.getvalue()
    assert "Initializing agent" in out or "search_code" in out
    assert "search_code" in out
    assert "腾讯 00700.HK" in out
