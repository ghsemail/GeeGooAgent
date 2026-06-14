"""OpenAI chat completions provider."""

from __future__ import annotations

import json
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


def _message_to_openai(message: Message) -> dict[str, Any]:
    payload: dict[str, Any] = {"role": message.role}
    if message.role == "assistant" and message.reasoning_content:
        payload["reasoning_content"] = message.reasoning_content
    if message.role == "assistant" and message.tool_calls:
        payload["content"] = message.content
        payload["tool_calls"] = [
            {
                "id": call.id,
                "type": "function",
                "function": {
                    "name": call.name,
                    "arguments": json.dumps(call.arguments, ensure_ascii=False),
                },
            }
            for call in message.tool_calls
        ]
        return payload
    if message.role == "tool":
        payload["content"] = message.content or ""
        payload["tool_call_id"] = message.tool_call_id or ""
        return payload
    payload["content"] = message.content or ""
    return payload


class OpenAIProvider:
    def __init__(
        self,
        model: str,
        api_key: str,
        *,
        base_url: str | None = None,
        create_fn: CreateFn | None = None,
        thinking_enabled: bool = False,
        reasoning_effort: str | None = None,
    ) -> None:
        self._model = model
        self._api_key = api_key
        self._base_url = base_url
        self._create_fn = create_fn
        self._thinking_enabled = thinking_enabled
        self._reasoning_effort = reasoning_effort

    @property
    def model(self) -> str:
        return self._model

    def _get_create(self) -> CreateFn:
        if self._create_fn is not None:
            return self._create_fn
        from openai import OpenAI

        kwargs: dict[str, str] = {"api_key": self._api_key}
        if self._base_url:
            kwargs["base_url"] = self._base_url
        client = OpenAI(**kwargs)
        return client.chat.completions.create

    def chat(
        self,
        messages: list[Message],
        tools: list[ToolSchema],
        *,
        temperature: float = 0.2,
        max_tokens: int = 4096,
    ) -> LLMResponse:
        payload_messages = [_message_to_openai(m) for m in messages]
        kwargs: dict[str, Any] = {
            "model": self._model,
            "messages": payload_messages,
            "temperature": temperature,
            "max_tokens": max_tokens,
        }
        if self._thinking_enabled:
            kwargs["extra_body"] = {"thinking": {"type": "enabled"}}
            if self._reasoning_effort:
                kwargs["reasoning_effort"] = self._reasoning_effort
        if tools:
            kwargs["tools"] = [
                {
                    "type": "function",
                    "function": {
                        "name": t.name,
                        "description": t.description,
                        "parameters": t.parameters,
                    },
                }
                for t in tools
            ]

        response = self._get_create()(**kwargs)
        return self._parse_response(response)

    def _parse_response(self, response: Any) -> LLMResponse:
        choice = response.choices[0]
        message = choice.message
        tool_calls: list[ToolCall] = []
        if getattr(message, "tool_calls", None):
            for tc in message.tool_calls:
                tool_calls.append(
                    ToolCall(
                        id=tc.id,
                        name=tc.function.name,
                        arguments=parse_tool_arguments(tc.function.arguments),
                    )
                )
        usage = response.usage
        token_usage = TokenUsage(
            prompt_tokens=int(getattr(usage, "prompt_tokens", 0) or 0),
            completion_tokens=int(getattr(usage, "completion_tokens", 0) or 0),
            model=self._model,
        )
        content = getattr(message, "content", None)
        reasoning = getattr(message, "reasoning_content", None)
        raw = response.model_dump() if hasattr(response, "model_dump") else None
        return LLMResponse(
            content=content,
            tool_calls=tool_calls,
            usage=token_usage,
            reasoning_content=reasoning,
            raw=raw,
        )
