"""Unit tests for OpenAIProvider."""

from __future__ import annotations

from types import SimpleNamespace

import pytest

from geegoo_agent.llm.openai_provider import OpenAIProvider
from geegoo_agent.llm.types import Message, ToolSchema


def _openai_tool_response() -> SimpleNamespace:
    return SimpleNamespace(
        choices=[
            SimpleNamespace(
                message=SimpleNamespace(
                    content=None,
                    tool_calls=[
                        SimpleNamespace(
                            id="call_1",
                            function=SimpleNamespace(
                                name="check_trading_day",
                                arguments='{"code": "00700.HK"}',
                            ),
                        )
                    ],
                )
            )
        ],
        usage=SimpleNamespace(prompt_tokens=12, completion_tokens=6),
        model_dump=lambda: {"id": "resp-1"},
    )


@pytest.mark.unit
def test_openai_provider_parses_tool_calls() -> None:
    provider = OpenAIProvider(
        "gpt-4o", "sk-test", create_fn=lambda **_kwargs: _openai_tool_response()
    )
    response = provider.chat(
        [Message(role="user", content="check")],
        [ToolSchema(name="check_trading_day", description="x", parameters={"type": "object"})],
    )
    assert len(response.tool_calls) == 1
    assert response.tool_calls[0].name == "check_trading_day"
    assert response.tool_calls[0].arguments["code"] == "00700.HK"
    assert response.usage.prompt_tokens == 12


@pytest.mark.unit
def test_openai_provider_accepts_custom_base_url() -> None:
    provider = OpenAIProvider(
        "deepseek-chat",
        "sk-test",
        base_url="https://api.deepseek.com",
        create_fn=lambda **_kwargs: SimpleNamespace(
            choices=[SimpleNamespace(message=SimpleNamespace(content="ok", tool_calls=None))],
            usage=SimpleNamespace(prompt_tokens=1, completion_tokens=1),
        ),
    )
    assert provider._base_url == "https://api.deepseek.com"


def test_openai_provider_text_response() -> None:
    text_resp = SimpleNamespace(
        choices=[SimpleNamespace(message=SimpleNamespace(content="done", tool_calls=None))],
        usage=SimpleNamespace(prompt_tokens=1, completion_tokens=2),
        model_dump=lambda: {},
    )
    provider = OpenAIProvider("gpt-4o", "sk-test", create_fn=lambda **_kwargs: text_resp)
    response = provider.chat([Message(role="user", content="hi")], [])
    assert response.content == "done"
    assert response.tool_calls == []


@pytest.mark.unit
def test_openai_provider_passes_tools_to_api() -> None:
    captured: dict = {}

    def capture(**kwargs):
        captured.update(kwargs)
        return _openai_tool_response()

    provider = OpenAIProvider("gpt-4o", "sk-test", create_fn=capture)
    provider.chat(
        [Message(role="user", content="x")],
        [ToolSchema(name="t1", description="d", parameters={"type": "object", "properties": {}})],
    )
    assert captured["model"] == "gpt-4o"
    assert captured["tools"][0]["function"]["name"] == "t1"
