"""Tests for DeepSeek thinking mode and plan display."""

from __future__ import annotations

from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest

from geegoo_agent.llm.openai_provider import OpenAIProvider
from geegoo_agent.llm.presets import resolve_thinking_enabled
from geegoo_agent.llm.types import Message
from geegoo_agent.runtime.react_loop import ReActLoop


@pytest.mark.unit
def test_resolve_thinking_auto_for_v4() -> None:
    assert resolve_thinking_enabled("deepseek", "deepseek-v4-flash", thinking=None) is True
    assert resolve_thinking_enabled("deepseek", "deepseek-chat", thinking=None) is False


@pytest.mark.unit
def test_openai_provider_parses_reasoning_content() -> None:
    captured: dict = {}

    def fake_create(**kwargs):
        captured.update(kwargs)
        return SimpleNamespace(
            choices=[
                SimpleNamespace(
                    message=SimpleNamespace(
                        content="最终答案",
                        reasoning_content="先分析再回答",
                        tool_calls=None,
                    )
                )
            ],
            usage=SimpleNamespace(prompt_tokens=3, completion_tokens=5),
            model_dump=lambda: {},
        )

    provider = OpenAIProvider(
        "deepseek-v4-pro",
        "sk-test",
        create_fn=fake_create,
        thinking_enabled=True,
        reasoning_effort="high",
    )
    response = provider.chat([Message(role="user", content="hi")], [])
    assert response.reasoning_content == "先分析再回答"
    assert captured.get("reasoning_effort") == "high"
    assert captured.get("extra_body") == {"thinking": {"type": "enabled"}}


@pytest.mark.unit
def test_react_loop_emits_llm_plan_event() -> None:
    events: list[str] = []

    class FakeProvider:
        model = "deepseek-v4-pro"

        def chat(self, messages, tools, *, temperature=0.2, max_tokens=4096):
            from geegoo_agent.llm.types import LLMResponse, TokenUsage

            return LLMResponse(
                content=None,
                reasoning_content="需要查腾讯代码",
                tool_calls=[],
                usage=TokenUsage(1, 1, "m"),
            )

    from geegoo_agent.llm.cost import CostManager
    from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
    from geegoo_agent.runtime.chat_session import ChatSession

    gateway = ModelGateway(FakeProvider(), CostManager(), GatewayConfig(max_retries=1))
    loop = ReActLoop(gateway, MagicMock(), on_progress=lambda e, d: events.append(e))
    session = ChatSession(id="chat-think")
    result = loop.run_turn(session, "查腾讯", MagicMock(), [])
    assert "llm_plan" in events
    assert "需要查腾讯" in result.step_records[0].summary
