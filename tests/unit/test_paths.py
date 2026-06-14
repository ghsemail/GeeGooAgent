"""Unit tests for default path resolution."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from geegoo_agent.paths import default_config_path, default_data_dir, geegoo_home


@pytest.mark.unit
def test_geegoo_home_default(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.delenv("GEEGOO_HOME", raising=False)
    monkeypatch.setattr("geegoo_agent.paths.Path.home", lambda: tmp_path)
    assert geegoo_home() == tmp_path / ".geegoo"


@pytest.mark.unit
def test_geegoo_home_env_override(monkeypatch, tmp_path: Path) -> None:
    custom = tmp_path / "custom-geegoo"
    monkeypatch.setenv("GEEGOO_HOME", str(custom))
    assert geegoo_home() == custom


@pytest.mark.unit
def test_default_config_path_env(monkeypatch, tmp_path: Path) -> None:
    cfg = tmp_path / "my-config.json"
    cfg.write_text("{}", encoding="utf-8")
    monkeypatch.setenv("GEEGOO_CONFIG", str(cfg))
    assert default_config_path() == cfg


@pytest.mark.unit
def test_default_config_path_prefers_home(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.delenv("GEEGOO_CONFIG", raising=False)
    home = tmp_path / ".geegoo"
    home.mkdir()
    home_cfg = home / "config.json"
    home_cfg.write_text("{}", encoding="utf-8")
    monkeypatch.setattr("geegoo_agent.paths.Path.home", lambda: tmp_path)
    monkeypatch.chdir(tmp_path)
    assert default_config_path() == home_cfg


@pytest.mark.unit
def test_default_data_dir(monkeypatch, tmp_path: Path) -> None:
    monkeypatch.setenv("GEEGOO_HOME", str(tmp_path / ".geegoo"))
    assert default_data_dir() == tmp_path / ".geegoo" / "data"
