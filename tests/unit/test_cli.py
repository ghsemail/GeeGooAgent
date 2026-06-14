"""Unit tests for CLI commands."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from geegoo_agent.cli import main
from geegoo_agent.memory.models import PreMarketWorking
from geegoo_agent.runtime.workflow import RunResult


@pytest.mark.unit
def test_cli_no_command_prints_help(capsys) -> None:
    code = main([])
    assert code == 0
    out = capsys.readouterr().out
    assert "geegoo" in out
    assert "update" in out


@pytest.mark.unit
@patch("geegoo_agent.cli.run_update", return_value=0)
def test_cli_update_command(mock_update: MagicMock) -> None:
    code = main(["update", "--method", "tarball"])
    assert code == 0
    mock_update.assert_called_once_with(
        method="tarball",
        branch="main",
        repo=None,
        skip_pip=False,
    )


@pytest.mark.unit
def test_cli_run_dry_run_exits_zero(sample_config_file: Path, capsys) -> None:
    code = main(["run", "pre_market", "--dry-run", "--config", str(sample_config_file)])
    out = capsys.readouterr().out
    assert code == 0
    assert "session=" in out
    assert "status=completed" in out


@pytest.mark.unit
def test_cli_run_missing_config_exits_two(tmp_path: Path) -> None:
    missing = tmp_path / "missing.json"
    code = main(["run", "pre_market", "--dry-run", "--config", str(missing)])
    assert code == 2


@pytest.mark.unit
def test_cli_resume_missing_session_exits_two(sample_config_file: Path, capsys) -> None:
    code = main(["resume", "--session", "sess-missing", "--config", str(sample_config_file)])
    assert code == 2
    assert "session not found" in capsys.readouterr().err


@pytest.mark.unit
def test_cli_resume_existing_session(sample_config_file: Path, capsys) -> None:
    run_code = main(["run", "pre_market", "--dry-run", "--config", str(sample_config_file)])
    assert run_code == 0
    session_id = capsys.readouterr().out.strip().split("session=")[1].split()[0]

    resume_code = main(
        ["resume", "--session", session_id, "--config", str(sample_config_file)]
    )
    out = capsys.readouterr().out
    assert resume_code == 0
    assert f"session={session_id}" in out
    assert "status=completed" in out


@pytest.mark.unit
def test_cli_run_unsupported_skill_exits_two(sample_config_file: Path, capsys) -> None:
    code = main(["run", "unknown_skill", "--dry-run", "--config", str(sample_config_file)])
    assert code == 2
    assert "unsupported skill" in capsys.readouterr().err.lower()


@pytest.mark.unit
@patch("geegoo_agent.cli.GeeGooApp.run_skill")
def test_cli_run_passes_mcp_token_via_env(
    mock_run: MagicMock,
    sample_config_file: Path,
    monkeypatch,
) -> None:
    monkeypatch.delenv("MCP_TOKEN", raising=False)
    mock_run.return_value = RunResult(
        session_id="sess-ok",
        status="completed",
        working=PreMarketWorking(session_id="sess-ok"),
    )
    import os

    main(
        [
            "run",
            "pre_market",
            "--dry-run",
            "--config",
            str(sample_config_file),
            "--mcp-token",
            "user-mcp-token-xyz",
        ]
    )
    assert os.environ.get("MCP_TOKEN") == "user-mcp-token-xyz"
    mock_run.assert_called_once()


@pytest.mark.unit
@patch("geegoo_agent.cli.GeeGooApp.run_skill")
def test_cli_run_failed_workflow_returns_one(
    mock_run: MagicMock,
    sample_config_file: Path,
) -> None:
    mock_run.return_value = RunResult(
        session_id="sess-fail",
        status="failed",
        working=PreMarketWorking(session_id="sess-fail"),
        last_error="boom",
    )
    code = main(["run", "pre_market", "--dry-run", "--config", str(sample_config_file)])
    assert code == 1
