"""File-based state store (L0)."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Protocol

from geegoo_agent.exceptions import StateStoreError


class StateStore(Protocol):
    def save(self, key: str, data: dict[str, Any]) -> None: ...
    def load(self, key: str) -> dict[str, Any] | None: ...
    def list_keys(self, prefix: str) -> list[str]: ...
    def delete(self, key: str) -> None: ...


class FileStateStore:
    """Persist JSON documents under a root directory using slash-separated keys."""

    def __init__(self, root: Path) -> None:
        self.root = Path(root)

    def _path_for(self, key: str) -> Path:
        if ".." in key or key.startswith("/"):
            raise StateStoreError(f"invalid state key: {key!r}")
        return self.root / f"{key}.json"

    def save(self, key: str, data: dict[str, Any]) -> None:
        path = self._path_for(key)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")

    def load(self, key: str) -> dict[str, Any] | None:
        path = self._path_for(key)
        if not path.exists():
            return None
        try:
            return json.loads(path.read_text(encoding="utf-8"))
        except json.JSONDecodeError as exc:
            raise StateStoreError(f"corrupt state file: {path}") from exc

    def list_keys(self, prefix: str) -> list[str]:
        base = self.root / Path(prefix) if prefix else self.root
        if not base.exists():
            return []
        keys: list[str] = []
        for path in base.rglob("*.json"):
            rel = path.relative_to(self.root).with_suffix("")
            keys.append(rel.as_posix())
        return sorted(keys)

    def delete(self, key: str) -> None:
        path = self._path_for(key)
        if path.exists():
            path.unlink()
