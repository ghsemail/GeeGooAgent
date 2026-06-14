"""Unit tests for CheckpointManager."""

from __future__ import annotations

import pytest

from geegoo_agent.exceptions import CheckpointError
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.state_store import FileStateStore


@pytest.fixture
def checkpoint_mgr(tmp_path):
    return CheckpointManager(FileStateStore(tmp_path))


@pytest.mark.unit
def test_save_and_load_latest(checkpoint_mgr: CheckpointManager) -> None:
    cp_id = checkpoint_mgr.save(
        session_id="sess-1",
        step=3,
        skill="pre_market",
        status="running",
        working={"phase": "A", "indices_done": True},
        last_tool="get_mcp_analysis",
    )
    assert cp_id == "cp-sess-1-0003"

    latest = checkpoint_mgr.load_latest("sess-1")
    assert latest is not None
    assert latest.step == 3
    assert latest.skill == "pre_market"
    assert latest.last_tool == "get_mcp_analysis"

    working = checkpoint_mgr.load_working(latest)
    assert working["phase"] == "A"


@pytest.mark.unit
def test_load_latest_returns_none_for_unknown_session(
    checkpoint_mgr: CheckpointManager,
) -> None:
    assert checkpoint_mgr.load_latest("missing") is None


@pytest.mark.unit
def test_list_returns_sorted_steps(checkpoint_mgr: CheckpointManager) -> None:
    checkpoint_mgr.save(
        session_id="sess-2",
        step=1,
        skill="pre_market",
        status="running",
        working={"n": 1},
    )
    checkpoint_mgr.save(
        session_id="sess-2",
        step=2,
        skill="pre_market",
        status="running",
        working={"n": 2},
    )
    steps = [cp.step for cp in checkpoint_mgr.list("sess-2")]
    assert steps == [1, 2]


@pytest.mark.unit
def test_latest_is_highest_step(checkpoint_mgr: CheckpointManager) -> None:
    checkpoint_mgr.save(
        session_id="sess-3",
        step=1,
        skill="pre_market",
        status="running",
        working={},
    )
    checkpoint_mgr.save(
        session_id="sess-3",
        step=5,
        skill="pre_market",
        status="running",
        working={"done": False},
    )
    latest = checkpoint_mgr.load_latest("sess-3")
    assert latest is not None
    assert latest.step == 5


@pytest.mark.unit
def test_load_working_missing_raises(checkpoint_mgr: CheckpointManager, tmp_path) -> None:
    store = FileStateStore(tmp_path)
    mgr = CheckpointManager(store)
    store.save(
        "checkpoint/sess-x/step-0001",
        {
            "checkpoint_id": "cp-sess-x-0001",
            "session_id": "sess-x",
            "step": 1,
            "status": "running",
            "skill": "pre_market",
            "working_key": "working/missing",
            "last_tool": None,
            "created_at": "2026-06-05T00:00:00+00:00",
        },
    )
    cp = mgr.load_latest("sess-x")
    assert cp is not None
    with pytest.raises(CheckpointError):
        mgr.load_working(cp)
