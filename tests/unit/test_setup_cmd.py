"""Unit tests for geegoo-agent setup."""

from __future__ import annotations

import json

import pytest

from geegoo_agent.paths import default_data_dir
from geegoo_agent.setup_cmd import run_setup


@pytest.mark.unit
def test_run_setup_non_interactive_writes_llm_and_mcp(tmp_path) -> None:
    config_path = tmp_path / "config.json"
    run_setup(
        config_path,
        project_root=tmp_path,
        provider="deepseek",
        token_key="sk-deepseek-test",
        mcp_token="mcp-test",
        interactive=False,
        skip_install=True,
    )
    raw = json.loads(config_path.read_text(encoding="utf-8"))
    assert raw["llm"]["provider"] == "deepseek"
    assert raw["llm"]["token_key"] == "sk-deepseek-test"
    assert raw["mcp_token"] == "mcp-test"
    assert "api_key_env" not in raw["llm"]
    assert raw["output_dir"] == str(default_data_dir())
