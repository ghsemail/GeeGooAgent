"""Shared pytest fixtures."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from geegoo_agent.config import AppConfig, LLMConfig, SandboxConfig


@pytest.fixture
def tmp_output_dir(tmp_path: Path) -> Path:
    """Isolated output directory for a test run."""
    out = tmp_path / "data"
    out.mkdir()
    return out


@pytest.fixture
def mock_mcp_token() -> str:
    return "test-mcp-token"


@pytest.fixture
def sample_config(tmp_output_dir: Path) -> AppConfig:
    return AppConfig(
        base_url="http://test:5700",
        api_key="sk-test-key",
        geegoo_url="http://test:5700",
        geegoo_api_key="sk-test-key",
        mcp_token="mcp-test-token",
        output_dir=tmp_output_dir,
        sandbox=SandboxConfig(allowed_hosts=["test"]),
        llm=LLMConfig(provider="openai", token_key="sk-test-llm-key"),
    )


@pytest.fixture
def sample_config_file(sample_config: AppConfig, tmp_path: Path) -> Path:
    path = tmp_path / "config.json"
    payload = sample_config.model_dump(mode="json")
    path.write_text(json.dumps(payload, indent=2), encoding="utf-8")
    return path
