"""Persisted interactive chat session."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any, Literal
from uuid import uuid4

from pydantic import BaseModel, Field

from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.llm.types import Message


class StepRecord(BaseModel):
    step: int
    timestamp: str
    kind: Literal["plan", "tool", "reply"]
    tool_name: str | None = None
    tool_status: str | None = None
    summary: str = ""
    tokens: int = 0


class ChatSession(BaseModel):
    id: str
    status: Literal["active", "closed"] = "active"
    created_at: str = Field(default_factory=lambda: datetime.now(UTC).isoformat())
    updated_at: str = Field(default_factory=lambda: datetime.now(UTC).isoformat())
    messages: list[dict[str, Any]] = Field(default_factory=list)
    step_records: list[StepRecord] = Field(default_factory=list)
    step_counter: int = 0

    def touch(self) -> None:
        self.updated_at = datetime.now(UTC).isoformat()

    def append_message(self, message: Message) -> None:
        payload: dict[str, Any] = {"role": message.role, "content": message.content}
        if message.reasoning_content:
            payload["reasoning_content"] = message.reasoning_content
        if message.tool_call_id:
            payload["tool_call_id"] = message.tool_call_id
        if message.tool_calls:
            payload["tool_calls"] = [
                {"id": c.id, "name": c.name, "arguments": c.arguments}
                for c in message.tool_calls
            ]
        self.messages.append(payload)
        self.touch()

    def to_llm_messages(self) -> list[Message]:
        restored: list[Message] = []
        for raw in self.messages:
            tool_calls = None
            if raw.get("tool_calls"):
                from geegoo_agent.llm.types import ToolCall

                tool_calls = [
                    ToolCall(id=item["id"], name=item["name"], arguments=item.get("arguments", {}))
                    for item in raw["tool_calls"]
                ]
            restored.append(
                Message(
                    role=raw["role"],
                    content=raw.get("content"),
                    tool_call_id=raw.get("tool_call_id"),
                    tool_calls=tool_calls,
                    reasoning_content=raw.get("reasoning_content"),
                )
            )
        return restored

    def add_step_record(self, record: StepRecord) -> None:
        self.step_records.append(record)
        self.touch()

    def tool_activity_summary(self) -> str:
        """Compact list of market-related tools already called in this chat."""
        tracked = {
            "search_code",
            "get_current_price",
            "get_ticker",
            "get_mcp_analysis",
            "get_capital_flow",
            "get_capital_distribution",
        }
        lines: list[str] = []
        for raw in self.messages:
            if raw.get("role") != "assistant":
                continue
            for call in raw.get("tool_calls") or []:
                name = call.get("name", "")
                if name not in tracked:
                    continue
                args = call.get("arguments") or {}
                if isinstance(args, str):
                    arg_text = args
                else:
                    parts = [f"{k}={v}" for k, v in args.items() if v not in (None, "", {}, [])]
                    arg_text = ", ".join(parts)
                lines.append(f"- {name}({arg_text})")
        return "\n".join(lines)


class ChatSessionStore:
    def __init__(self, store: FileStateStore) -> None:
        self._store = store

    def _key(self, session_id: str) -> str:
        return f"chat/{session_id}"

    def create(self) -> ChatSession:
        session = ChatSession(id=f"chat-{uuid4().hex[:12]}")
        self.save(session)
        return session

    def load(self, session_id: str) -> ChatSession | None:
        data = self._store.load(self._key(session_id))
        if data is None:
            return None
        return ChatSession.model_validate(data)

    def save(self, session: ChatSession) -> None:
        session.touch()
        self._store.save(self._key(session.id), session.model_dump())
