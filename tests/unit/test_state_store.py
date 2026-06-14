"""Unit tests for FileStateStore."""

from __future__ import annotations

import pytest

from geegoo_agent.exceptions import StateStoreError
from geegoo_agent.infra.state_store import FileStateStore


@pytest.mark.unit
def test_save_and_load_roundtrip(tmp_path) -> None:
    store = FileStateStore(tmp_path)
    store.save("session/sess-1", {"step": 1, "status": "running"})
    loaded = store.load("session/sess-1")
    assert loaded == {"step": 1, "status": "running"}


@pytest.mark.unit
def test_load_missing_key_returns_none(tmp_path) -> None:
    store = FileStateStore(tmp_path)
    assert store.load("missing/key") is None


@pytest.mark.unit
def test_list_keys_with_prefix(tmp_path) -> None:
    store = FileStateStore(tmp_path)
    store.save("working/a", {"x": 1})
    store.save("working/b", {"x": 2})
    store.save("session/c", {"x": 3})

    assert store.list_keys("working") == ["working/a", "working/b"]


@pytest.mark.unit
def test_delete_removes_file(tmp_path) -> None:
    store = FileStateStore(tmp_path)
    store.save("session/x", {"a": 1})
    store.delete("session/x")
    assert store.load("session/x") is None


@pytest.mark.unit
def test_invalid_key_rejected(tmp_path) -> None:
    store = FileStateStore(tmp_path)
    with pytest.raises(StateStoreError):
        store.save("../etc/passwd", {"bad": True})


@pytest.mark.unit
def test_corrupt_json_raises(tmp_path) -> None:
    store = FileStateStore(tmp_path)
    path = tmp_path / "broken.json"
    path.write_text("{not json", encoding="utf-8")
    with pytest.raises(StateStoreError):
        store.load("broken")
