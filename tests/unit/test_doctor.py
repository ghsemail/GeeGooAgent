"""Unit tests for geegoo doctor."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from geegoo_agent.doctor_cmd import run_doctor
from geegoo_agent.llm.types import LLMResponse, TokenUsage


@pytest.mark.unit
def test_doctor_missing_config(tmp_path: Path, capsys) -> None:
    missing = tmp_path / "missing.json"
    code = run_doctor(missing)
    out = capsys.readouterr().out
    assert code == 1
    assert "[FAIL] config" in out


@pytest.mark.unit
def test_doctor_placeholder_secrets_fail(tmp_path: Path, capsys) -> None:
    bad = tmp_path / "config.json"
    bad.write_text(
        """{
  "base_url": "http://test:5700",
  "api_key": "sk-REPLACE",
  "geegoo_url": "http://test:5700",
  "geegoo_api_key": "sk-REPLACE",
  "mcp_token": "",
  "output_dir": "./data",
  "sandbox": {"allowed_hosts": ["test"]},
  "llm": {"provider": "openai", "token_key": ""}
}""",
        encoding="utf-8",
    )
    code = run_doctor(bad, skip_api=True, skip_llm=True)
    out = capsys.readouterr().out
    assert code == 1
    assert "[OK] config" in out
    assert "[FAIL]" in out
    assert "geegoo setup" in out


@pytest.mark.unit
@patch("geegoo_agent.doctor_cmd._post_json")
def test_doctor_api_checks_ok(
    mock_post: MagicMock,
    sample_config_file: Path,
    capsys,
) -> None:
    mock_post.return_value = (200, '{"ok":true}')
    code = run_doctor(sample_config_file, skip_llm=True)
    out = capsys.readouterr().out
    assert code == 0
    assert "[OK] geegoo mcp checkTradingDay" in out
    assert "[OK] geegoo mcp getCurrentPrice" in out
    assert mock_post.call_count == 3


@pytest.mark.unit
@patch("geegoo_agent.doctor_cmd.ModelGateway")
@patch("geegoo_agent.doctor_cmd._post_json")
def test_doctor_full_pass(
    mock_post: MagicMock,
    mock_gateway_cls: MagicMock,
    sample_config_file: Path,
    capsys,
) -> None:
    mock_post.return_value = (200, '{"ok":true}')
    gateway = MagicMock()
    gateway.chat.return_value = LLMResponse(
        content="ok",
        tool_calls=[],
        usage=TokenUsage(prompt_tokens=1, completion_tokens=1, model="test"),
    )
    mock_gateway_cls.return_value = gateway

    code = run_doctor(sample_config_file)
    out = capsys.readouterr().out
    assert code == 0
    assert "全部检查通过" in out
    assert "[OK] LLM" in out


@pytest.mark.unit
@patch("geegoo_agent.doctor_cmd._post_json")
def test_cli_doctor_command(
    mock_post: MagicMock,
    sample_config_file: Path,
    capsys,
) -> None:
    from geegoo_agent.cli import main

    mock_post.return_value = (200, '{"ok":true}')
    code = main(
        [
            "doctor",
            "--config",
            str(sample_config_file),
            "--skip-llm",
        ]
    )
    out = capsys.readouterr().out
    assert code == 0
    assert "[OK] config" in out
