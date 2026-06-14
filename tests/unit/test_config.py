"""Unit tests for configuration loading."""

from __future__ import annotations

import json

import pytest

from geegoo_agent.config import AppConfig, load_config
from geegoo_agent.exceptions import ConfigError


@pytest.fixture
def valid_config_dict() -> dict:
    return {
        "base_url": "http://118.195.135.97:5700",
        "api_key": "sk-test-key",
        "geegoo_url": "http://118.195.135.97:5700",
        "geegoo_api_key": "sk-test-key",
        "mcp_token": "mcp-test-token",
        "output_dir": "./data",
        "sandbox": {"allowed_hosts": ["118.195.135.97"]},
    }


@pytest.mark.unit
def test_load_config_success(tmp_path, valid_config_dict) -> None:
    path = tmp_path / "config.json"
    path.write_text(json.dumps(valid_config_dict), encoding="utf-8")
    config = load_config(path)
    assert config.base_url.endswith(":5700")
    assert config.mcp_token == "mcp-test-token"
    assert config.workspace_root.exists()


@pytest.mark.unit
def test_load_config_missing_file_raises(tmp_path) -> None:
    with pytest.raises(ConfigError, match="not found"):
        load_config(tmp_path / "missing.json")


@pytest.mark.unit
def test_load_config_invalid_json_raises(tmp_path) -> None:
    path = tmp_path / "bad.json"
    path.write_text("{", encoding="utf-8")
    with pytest.raises(ConfigError, match="invalid JSON"):
        load_config(path)


@pytest.mark.unit
def test_load_config_missing_required_field_raises(tmp_path, valid_config_dict) -> None:
    del valid_config_dict["api_key"]
    path = tmp_path / "config.json"
    path.write_text(json.dumps(valid_config_dict), encoding="utf-8")
    with pytest.raises(ConfigError, match="invalid config"):
        load_config(path)


@pytest.mark.unit
def test_app_config_defaults() -> None:
    config = AppConfig(
        base_url="http://x:5700",
        api_key="sk",
        geegoo_url="http://x:5700",
        geegoo_api_key="sk",
        mcp_token="t",
    )
    assert config.llm.provider == "openai"
    assert config.max_steps == 80
