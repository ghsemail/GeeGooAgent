"""LLM provider protocol."""

from __future__ import annotations

from typing import Protocol

from geegoo_agent.llm.types import LLMResponse, Message, ToolSchema


class LLMProvider(Protocol):
    @property
    def model(self) -> str: ...

    def chat(
        self,
        messages: list[Message],
        tools: list[ToolSchema],
        *,
        temperature: float,
        max_tokens: int,
    ) -> LLMResponse: ...
