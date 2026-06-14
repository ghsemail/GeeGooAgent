"""Unit tests for AnthropicProvider."""

from __future__ import annotations

from types import SimpleNamespace

import pytest

from geegoo_agent.llm.anthropic_provider import AnthropicProvider
from geegoo_agent.llm.types import Message, ToolSchema


def _anthropic_tool_response() -> SimpleNamespace:
    return SimpleNamespace(
        content=[
            SimpleNamespace(
                type="tool_use", id="tu_1", name="get_mcp_analysis", input={"period": "weekly"}
            ),
        ],
        usage=SimpleNamespace(input_tokens=20, output_tokens=8),
        model_dump=lambda: {},
    )


@pytest.mark.unit
def test_anthropic_provider_parses_tool_use() -> None:
    provider = AnthropicProvider(
        "claude-sonnet-4-20250514",
        "sk-ant",
        create_fn=lambda **_kwargs: _anthropic_tool_response(),
    )
    response = provider.chat(
        [Message(role="user", content="analyze")],
        [ToolSchema(name="get_mcp_analysis", description="x", parameters={})],
    )
    assert response.tool_calls[0].name == "get_mcp_analysis"
    assert response.tool_calls[0].arguments["period"] == "weekly"
    assert response.usage.prompt_tokens == 20


@pytest.mark.unit
def test_anthropic_provider_extracts_system_prompt() -> None:
    captured: dict = {}

    def capture(**kwargs):
        captured.update(kwargs)
        return SimpleNamespace(
            content=[SimpleNamespace(type="text", text="ok")],
            usage=SimpleNamespace(input_tokens=1, output_tokens=1),
            model_dump=lambda: {},
        )

    provider = AnthropicProvider("claude", "key", create_fn=capture)
    provider.chat(
        [
            Message(role="system", content="You are helpful"),
            Message(role="user", content="hi"),
        ],
        [],
    )
    assert captured["system"] == "You are helpful"
    assert captured["messages"] == [{"role": "user", "content": "hi"}]
