"""Model gateway with retry and optional fallback."""

from __future__ import annotations

import time
from collections.abc import Callable
from dataclasses import dataclass

from geegoo_agent.exceptions import ModelGatewayError
from geegoo_agent.llm.base import LLMProvider
from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.types import LLMResponse, Message, ToolSchema


@dataclass
class GatewayConfig:
    max_retries: int = 3
    retry_wait_seconds: float = 5.0
    temperature: float = 0.2
    max_tokens: int = 4096


class ModelGateway:
    def __init__(
        self,
        primary: LLMProvider,
        cost: CostManager,
        config: GatewayConfig | None = None,
        *,
        fallback: LLMProvider | None = None,
        sleeper: Callable[[float], None] = time.sleep,
    ) -> None:
        self._primary = primary
        self._fallback = fallback
        self._cost = cost
        self._config = config or GatewayConfig()
        self._sleep = sleeper

    def chat(
        self,
        messages: list[Message],
        tools: list[ToolSchema],
        *,
        session_id: str = "default",
        step: int = 0,
    ) -> LLMResponse:
        last_error: Exception | None = None
        for attempt in range(self._config.max_retries):
            try:
                response = self._invoke(self._primary, messages, tools)
                self._cost.record(session_id, step, response.usage)
                return response
            except Exception as exc:
                last_error = exc
                if attempt < self._config.max_retries - 1:
                    self._sleep(self._config.retry_wait_seconds)

        if self._fallback is not None:
            try:
                response = self._invoke(self._fallback, messages, tools)
                self._cost.record(session_id, step, response.usage)
                return response
            except Exception as exc:
                last_error = exc

        raise ModelGatewayError(f"LLM gateway failed: {last_error}") from last_error

    def _invoke(
        self,
        provider: LLMProvider,
        messages: list[Message],
        tools: list[ToolSchema],
    ) -> LLMResponse:
        return provider.chat(
            messages,
            tools,
            temperature=self._config.temperature,
            max_tokens=self._config.max_tokens,
        )
