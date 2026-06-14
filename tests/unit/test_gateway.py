"""Unit tests for ModelGateway."""

from __future__ import annotations

import pytest

from geegoo_agent.exceptions import ModelGatewayError
from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
from geegoo_agent.llm.types import LLMResponse, Message, TokenUsage


class FakeProvider:
    def __init__(self, model: str, responses: list[LLMResponse | Exception]) -> None:
        self.model = model
        self._responses = list(responses)
        self.calls = 0

    def chat(self, messages, tools, *, temperature, max_tokens) -> LLMResponse:
        self.calls += 1
        item = self._responses.pop(0)
        if isinstance(item, Exception):
            raise item
        return item


def _ok_response(model: str = "gpt-4o") -> LLMResponse:
    return LLMResponse(
        content="ok",
        tool_calls=[],
        usage=TokenUsage(prompt_tokens=5, completion_tokens=3, model=model),
    )


@pytest.mark.unit
def test_gateway_returns_primary_response() -> None:
    primary = FakeProvider("gpt-4o", [_ok_response()])
    gateway = ModelGateway(primary, CostManager(), GatewayConfig(max_retries=1))
    response = gateway.chat([Message(role="user", content="hi")], [])
    assert response.content == "ok"
    assert primary.calls == 1


@pytest.mark.unit
def test_gateway_retries_then_succeeds() -> None:
    primary = FakeProvider(
        "gpt-4o",
        [RuntimeError("timeout"), RuntimeError("timeout"), _ok_response()],
    )
    sleeps: list[float] = []
    gateway = ModelGateway(
        primary,
        CostManager(),
        GatewayConfig(max_retries=3, retry_wait_seconds=0.01),
        sleeper=sleeps.append,
    )
    response = gateway.chat([Message(role="user", content="hi")], [])
    assert response.content == "ok"
    assert primary.calls == 3
    assert len(sleeps) == 2


@pytest.mark.unit
def test_gateway_uses_fallback_after_primary_exhausted() -> None:
    primary = FakeProvider("gpt-4o", [RuntimeError("fail")] * 3)
    fallback = FakeProvider("claude", [_ok_response("claude")])
    gateway = ModelGateway(
        primary,
        CostManager(),
        GatewayConfig(max_retries=3, retry_wait_seconds=0),
        fallback=fallback,
        sleeper=lambda _s: None,
    )
    response = gateway.chat([Message(role="user", content="hi")], [])
    assert response.usage.model == "claude"
    assert primary.calls == 3
    assert fallback.calls == 1


@pytest.mark.unit
def test_gateway_raises_when_all_providers_fail() -> None:
    primary = FakeProvider("gpt-4o", [RuntimeError("p")] * 3)
    fallback = FakeProvider("claude", [RuntimeError("f")])
    gateway = ModelGateway(
        primary,
        CostManager(),
        GatewayConfig(max_retries=2, retry_wait_seconds=0),
        fallback=fallback,
        sleeper=lambda _s: None,
    )
    with pytest.raises(ModelGatewayError, match="LLM gateway failed"):
        gateway.chat([Message(role="user", content="hi")], [])


@pytest.mark.unit
def test_gateway_records_cost() -> None:
    primary = FakeProvider("gpt-4o", [_ok_response()])
    cost = CostManager()
    gateway = ModelGateway(primary, cost, GatewayConfig(max_retries=1))
    gateway.chat([Message(role="user", content="hi")], [], session_id="sess-9", step=2)
    total = cost.session_total("sess-9")
    assert total.prompt_tokens == 5
    assert total.completion_tokens == 3
