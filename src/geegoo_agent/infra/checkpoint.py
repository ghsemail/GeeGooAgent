"""Step-level checkpoint manager (L0)."""

from __future__ import annotations

from dataclasses import asdict, dataclass
from datetime import UTC, datetime
from typing import Any

from geegoo_agent.exceptions import CheckpointError
from geegoo_agent.infra.state_store import FileStateStore


@dataclass
class Checkpoint:
    checkpoint_id: str
    session_id: str
    step: int
    status: str
    skill: str
    working_key: str
    last_tool: str | None
    created_at: str

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> Checkpoint:
        return cls(
            checkpoint_id=data["checkpoint_id"],
            session_id=data["session_id"],
            step=int(data["step"]),
            status=data["status"],
            skill=data["skill"],
            working_key=data["working_key"],
            last_tool=data.get("last_tool"),
            created_at=data["created_at"],
        )

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


class CheckpointManager:
    def __init__(self, store: FileStateStore) -> None:
        self._store = store

    def _meta_key(self, session_id: str, step: int) -> str:
        return f"checkpoint/{session_id}/step-{step:04d}"

    def save(
        self,
        *,
        session_id: str,
        step: int,
        skill: str,
        status: str,
        working: dict[str, Any],
        last_tool: str | None = None,
    ) -> str:
        working_key = f"working/{session_id}"
        self._store.save(working_key, working)

        checkpoint_id = f"cp-{session_id}-{step:04d}"
        record = Checkpoint(
            checkpoint_id=checkpoint_id,
            session_id=session_id,
            step=step,
            status=status,
            skill=skill,
            working_key=working_key,
            last_tool=last_tool,
            created_at=datetime.now(UTC).isoformat(),
        )
        self._store.save(self._meta_key(session_id, step), record.to_dict())
        return checkpoint_id

    def load_latest(self, session_id: str) -> Checkpoint | None:
        keys = [
            k
            for k in self._store.list_keys(f"checkpoint/{session_id}")
            if k.startswith(f"checkpoint/{session_id}/step-")
        ]
        if not keys:
            return None
        latest_key = sorted(keys)[-1]
        data = self._store.load(latest_key)
        if data is None:
            return None
        try:
            return Checkpoint.from_dict(data)
        except KeyError as exc:
            raise CheckpointError(f"invalid checkpoint record: {latest_key}") from exc

    def load_working(self, checkpoint: Checkpoint) -> dict[str, Any]:
        data = self._store.load(checkpoint.working_key)
        if data is None:
            raise CheckpointError(f"missing working state: {checkpoint.working_key}")
        return data

    def list(self, session_id: str) -> list[Checkpoint]:
        keys = [
            k
            for k in self._store.list_keys(f"checkpoint/{session_id}")
            if k.startswith(f"checkpoint/{session_id}/step-")
        ]
        results: list[Checkpoint] = []
        for key in sorted(keys):
            data = self._store.load(key)
            if data is not None:
                results.append(Checkpoint.from_dict(data))
        return results
