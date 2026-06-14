"""Tests for geegoo setup install step."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import pytest

from geegoo_agent.setup_cmd import ensure_installed


@pytest.mark.unit
def test_ensure_installed_runs_pip_when_pyproject_present(tmp_path: Path) -> None:
    (tmp_path / "pyproject.toml").write_text("[project]\nname='x'\n", encoding="utf-8")
    calls: list[list[str]] = []

    def fake_check_call(cmd, cwd=None, **_kwargs):
        calls.append(cmd)

    with patch("geegoo_agent.setup_cmd.subprocess.check_call", side_effect=fake_check_call):
        ensure_installed(tmp_path, dev=True)
    assert len(calls) == 1
    assert calls[0][1:4] == ["-m", "pip", "install"]
    assert "-e" in calls[0]
    assert calls[0][-1] in {".", ".[dev]"}
