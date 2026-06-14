"""Unit tests for GitHub PAT setup and persistence."""

from __future__ import annotations

import json
import os

import pytest

from geegoo_agent.setup_cmd import run_setup
from geegoo_agent.update_cmd import github_token, github_token_path, save_github_token


@pytest.mark.unit
def test_save_github_token_writes_file(tmp_path, monkeypatch) -> None:
    monkeypatch.setenv("GEEGOO_HOME", str(tmp_path))
    path = save_github_token("ghp_test_token_12345678")
    assert path == tmp_path / "github_token"
    assert path.read_text(encoding="utf-8").strip() == "ghp_test_token_12345678"
    if os.name != "nt":
        assert oct(path.stat().st_mode & 0o777) == oct(0o600)
    assert github_token() == "ghp_test_token_12345678"


@pytest.mark.unit
def test_run_setup_writes_github_token(tmp_path, monkeypatch) -> None:
    monkeypatch.setenv("GEEGOO_HOME", str(tmp_path))
    config_path = tmp_path / "config.json"
    run_setup(
        config_path,
        project_root=tmp_path,
        provider="deepseek",
        token_key="sk-deepseek-test",
        mcp_token="mcp-test",
        github_token_value="ghp_setup_token",
        interactive=False,
        skip_install=True,
    )
    raw = json.loads(config_path.read_text(encoding="utf-8"))
    assert raw["mcp_token"] == "mcp-test"
    assert github_token_path().read_text(encoding="utf-8").strip() == "ghp_setup_token"
