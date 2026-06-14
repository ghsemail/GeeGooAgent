"""Session lifecycle management."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Literal
from uuid import uuid4

from pydantic import BaseModel, Field

from geegoo_agent.infra.state_store import FileStateStore

SessionStatus = Literal["created", "running", "paused", "completed", "failed"]


class Session(BaseModel):
    id: str
    skill_name: str
    status: SessionStatus = "created"
    step: int = 0
    created_at: str = Field(default_factory=lambda: datetime.now(UTC).isoformat())
    updated_at: str = Field(default_factory=lambda: datetime.now(UTC).isoformat())
    error: str | None = None

    def touch(self) -> None:
        self.updated_at = datetime.now(UTC).isoformat()

    def mark_running(self) -> None:
        self.status = "running"
        self.touch()

    def mark_completed(self) -> None:
        self.status = "completed"
        self.touch()

    def mark_failed(self, error: str) -> None:
        self.status = "failed"
        self.error = error
        self.touch()


class SessionManager:
    def __init__(self, store: FileStateStore) -> None:
        self._store = store

    def _key(self, session_id: str) -> str:
        return f"session/{session_id}"

    def create(self, skill_name: str) -> Session:
        session = Session(id=f"sess-{uuid4().hex[:12]}", skill_name=skill_name)
        self.save(session)
        return session

    def load(self, session_id: str) -> Session | None:
        data = self._store.load(self._key(session_id))
        if data is None:
            return None
        return Session.model_validate(data)

    def save(self, session: Session) -> None:
        session.touch()
        self._store.save(self._key(session.id), session.model_dump())
