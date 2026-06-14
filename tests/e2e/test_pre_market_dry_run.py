"""End-to-end dry-run test for the full pre-market workflow."""

from __future__ import annotations

import re
from datetime import date
from pathlib import Path

import pytest

from geegoo_agent.cli import main
from geegoo_agent.config import load_config
from geegoo_agent.infra.checkpoint import CheckpointManager
from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.runtime.pre_market_constants import DRY_RUN_SAMPLE_BOTS
from geegoo_agent.runtime.pre_market_workflow import (
    PRE_MARKET_PER_STOCK_STEPS,
    PRE_MARKET_PHASE_A_STEPS,
)
from geegoo_agent.supervisor.pre_market import run_pre_market_supervisor

PROJECT_ROOT = Path(__file__).resolve().parents[2]

PHASE_A_STEP_NAMES = [
    step.name for step in PRE_MARKET_PHASE_A_STEPS if step.tool != "write_execution_log"
]
PHASE_B_STEP_NAMES = [
    step.name for step in PRE_MARKET_PER_STOCK_STEPS if step.tool != "write_execution_log"
]
EXPECTED_LOG_STEPS = [
    *PHASE_A_STEP_NAMES,
    "phase_a_complete",
    *[f"00700.HK/{name}" for name in PHASE_B_STEP_NAMES],
    "stock_complete:00700.HK",
    "supervisor",
]


def _parse_session_id(cli_output: str) -> str:
    match = re.search(r"session=(\S+)", cli_output)
    assert match, f"session id not found in CLI output: {cli_output!r}"
    return match.group(1)


@pytest.mark.e2e
def test_pre_market_dry_run_full(sample_config_file: Path, httpx_mock, capsys) -> None:
    exit_code = main(
        ["run", "pre_market", "--dry-run", "--config", str(sample_config_file)]
    )
    captured = capsys.readouterr()
    assert exit_code == 0, captured.err
    assert "status=completed" in captured.out

    session_id = _parse_session_id(captured.out)
    config = load_config(sample_config_file)
    workspace = config.workspace_root
    today = date.today().isoformat()

    log_path = workspace / today / "execution-log.md"
    assert log_path.exists(), "execution-log.md should exist"
    log_content = log_path.read_text(encoding="utf-8")
    for step_name in EXPECTED_LOG_STEPS:
        assert step_name in log_content, f"missing step in execution-log: {step_name}"
    log_lines = [line for line in log_content.splitlines() if line.startswith("- [")]
    assert len(log_lines) >= len(EXPECTED_LOG_STEPS)

    store = FileStateStore(workspace)
    checkpoint_mgr = CheckpointManager(store)
    latest = checkpoint_mgr.load_latest(session_id)
    assert latest is not None, "final checkpoint should exist"
    expected_final_step = len(PRE_MARKET_PHASE_A_STEPS) + len(PRE_MARKET_PER_STOCK_STEPS)
    assert latest.step == expected_final_step

    working = WorkingMemoryStore(store).load(session_id)
    assert working is not None
    assert working.phase == "done"
    assert working.is_trading_day is True
    assert working.market_context.indices_done is True
    assert working.market_context.market_news_done is True

    for bot in DRY_RUN_SAMPLE_BOTS:
        code = bot["code"]
        assert code in working.stocks
        assert working.stocks[code].status == "reported"
        assert working.stocks[code].report_id == "dry-run-id"
        report_path = workspace / "reports" / today / f"{code}-premarket.md"
        assert report_path.exists(), f"missing local report for {code}"
        assert report_path.read_text(encoding="utf-8").strip()

    create_requests = [
        req
        for req in httpx_mock.get_requests()
        if "createPreMarketReport" in str(req.url)
    ]
    assert create_requests == [], "dry-run must not POST createPreMarketReport"
    assert httpx_mock.get_requests() == [], "dry-run must not make HTTP calls"

    sup = run_pre_market_supervisor(
        working,
        workspace_root=workspace,
        project_root=PROJECT_ROOT,
    )
    assert sup.ok, sup.summary


@pytest.mark.e2e
def test_pre_market_dry_run_resume_after_complete(sample_config_file: Path, capsys) -> None:
    """Resume on an already-completed session remains idempotent and passes supervisor."""
    first_code = main(
        ["run", "pre_market", "--dry-run", "--config", str(sample_config_file)]
    )
    first_out = capsys.readouterr().out
    assert first_code == 0
    session_id = _parse_session_id(first_out)

    resume_code = main(
        ["resume", "--session", session_id, "--config", str(sample_config_file)]
    )
    resume_out = capsys.readouterr().out
    assert resume_code == 0, resume_out
    assert "status=completed" in resume_out

    config = load_config(sample_config_file)
    store = FileStateStore(config.workspace_root)
    working = WorkingMemoryStore(store).load(session_id)
    assert working is not None
    assert working.phase == "done"

    sup = run_pre_market_supervisor(
        working,
        workspace_root=config.workspace_root,
        project_root=PROJECT_ROOT,
    )
    assert sup.ok
