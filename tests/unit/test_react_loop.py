"""Unit tests for chat ReAct loop."""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
from geegoo_agent.llm.types import LLMResponse, Message, TokenUsage, ToolCall, ToolSchema
from geegoo_agent.runtime.chat_session import ChatSession
from geegoo_agent.runtime.react_loop import ReActLoop
from geegoo_agent.tools.types import ToolCallRequest, ToolContext, ToolResult


class FakeProvider:
    def __init__(self, responses: list[LLMResponse]) -> None:
        self._responses = list(responses)
        self.model = "gpt-test"

    def chat(self, messages, tools, *, temperature=0.2, max_tokens=4096):
        return self._responses.pop(0)


@pytest.mark.unit
def test_react_loop_executes_tool_then_replies(tmp_path, sample_config) -> None:
    provider = FakeProvider(
        [
            LLMResponse(
                content=None,
                tool_calls=[ToolCall(id="c1", name="search_code", arguments={"regex": "腾讯"})],
                usage=TokenUsage(10, 5, "gpt-test"),
            ),
            LLMResponse(
                content="腾讯控股代码 00700.HK。",
                tool_calls=[],
                usage=TokenUsage(8, 12, "gpt-test"),
            ),
        ]
    )
    gateway = ModelGateway(provider, CostManager(), GatewayConfig(max_retries=1))
    executor = MagicMock()
    executor.execute.return_value = ToolResult(
        status="ok",
        summary="search_code: 1 item(s)",
        data={"items": [{"code": "00700.HK"}]},
    )
    session = ChatSession(id="chat-test")
    loop = ReActLoop(gateway, executor, max_tool_rounds=4)
    ctx = MagicMock(spec=ToolContext)

    result = loop.run_turn(
        session,
        "查一下腾讯",
        ctx,
        [ToolSchema(name="search_code", description="search", parameters={})],
    )

    assert result.assistant_text == "腾讯控股代码 00700.HK。"
    assert not result.failed
    executor.execute.assert_called_once()
    call = executor.execute.call_args[0][0]
    assert isinstance(call, ToolCallRequest)
    assert call.name == "search_code"
