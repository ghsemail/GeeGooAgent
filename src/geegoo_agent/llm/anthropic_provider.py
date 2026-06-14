"""Anthropic messages API provider."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from geegoo_agent.llm.types import (
    LLMResponse,
    Message,
    TokenUsage,
    ToolCall,
    ToolSchema,
    parse_tool_arguments,
)

CreateFn = Callable[..., Any]


class AnthropicProvider:
    def __init__(
        self,
        model: str,
        api_key: str,
        *,
        create_fn: CreateFn | None = None,
    ) -> None:
        self._model = model
        self._api_key = api_key
        self._create_fn = create_fn

    @property
    def model(self) -> str:
        return self._model

    def _get_create(self) -> CreateFn:
        if self._create_fn is not None:
            return self._create_fn
        from anthropic import Anthropic

        client = Anthropic(api_key=self._api_key)
        return client.messages.create

    def chat(
        self,
        messages: list[Message],
        tools: list[ToolSchema],
        *,
        temperature: float = 0.2,
        max_tokens: int = 4096,
    ) -> LLMResponse:
        system_parts = [m.content for m in messages if m.role == "system"]
        system = "\n".join(system_parts) if system_parts else None
        api_messages = [
            {"role": m.role, "content": m.content}
            for m in messages
            if m.role in {"user", "assistant"}
        ]
        kwargs: dict[str, Any] = {
            "model": self._model,
            "messages": api_messages,
            "max_tokens": max_tokens,
            "temperature": temperature,
        }
        if system:
            kwargs["system"] = system
        if tools:
            kwargs["tools"] = [
                {
                    "name": t.name,
                    "description": t.description,
                    "input_schema": t.parameters,
                }
                for t in tools
            ]

        response = self._get_create()(**kwargs)
        return self._parse_response(response)

    def _parse_response(self, response: Any) -> LLMResponse:
        content_parts: list[str] = []
        tool_calls: list[ToolCall] = []
        for block in response.content:
            block_type = getattr(block, "type", None)
            if block_type == "text":
                content_parts.append(block.text)
            elif block_type == "tool_use":
                tool_calls.append(
                    ToolCall(
                        id=block.id,
                        name=block.name,
                        arguments=parse_tool_arguments(block.input),
                    )
                )
        usage = response.usage
        token_usage = TokenUsage(
            prompt_tokens=int(getattr(usage, "input_tokens", 0) or 0),
            completion_tokens=int(getattr(usage, "output_tokens", 0) or 0),
            model=self._model,
        )
        content = "\n".join(content_parts) if content_parts else None
        raw = response.model_dump() if hasattr(response, "model_dump") else None
        return LLMResponse(content=content, tool_calls=tool_calls, usage=token_usage, raw=raw)
