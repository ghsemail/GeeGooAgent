"""Unit tests for geegoo update."""

from __future__ import annotations

import io
import tarfile
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from geegoo_agent.update_cmd import (
    merge_tarball_into,
    resolve_install_dir,
    run_update,
    sync_source,
)


@pytest.mark.unit
def test_resolve_install_dir_prefers_geegoo_home(tmp_path, monkeypatch) -> None:
    home = tmp_path / ".geegoo" / "geegoo-agent"
    home.mkdir(parents=True)
    (home / "pyproject.toml").write_text("[project]\nname='x'\n", encoding="utf-8")
    (home / "src" / "geegoo").mkdir(parents=True)
    monkeypatch.setenv("GEEGOO_HOME", str(tmp_path / ".geegoo"))
    monkeypatch.delenv("GEEGOO_INSTALL_DIR", raising=False)
    assert resolve_install_dir() == home


@pytest.mark.unit
def test_merge_tarball_preserves_venv(tmp_path) -> None:
    install_dir = tmp_path / "install"
    install_dir.mkdir()
    (install_dir / "old.txt").write_text("old", encoding="utf-8")
    venv = install_dir / "venv"
    venv.mkdir()
    (venv / "keep.txt").write_text("venv", encoding="utf-8")

    payload_root = tmp_path / "GeeGooAgent-main"
    payload_root.mkdir()
    (payload_root / "new.txt").write_text("new", encoding="utf-8")
    (payload_root / "src").mkdir()

    archive = tmp_path / "pkg.tar.gz"
    with tarfile.open(archive, "w:gz") as tar:
        tar.add(payload_root, arcname="GeeGooAgent-main")

    merge_tarball_into(install_dir, archive)

    assert not (install_dir / "old.txt").exists()
    assert (install_dir / "new.txt").read_text(encoding="utf-8") == "new"
    assert (install_dir / "venv" / "keep.txt").read_text(encoding="utf-8") == "venv"


@pytest.mark.unit
@patch("geegoo_agent.update_cmd._sync_via_tarball", return_value="tarball (test)")
@patch("geegoo_agent.update_cmd.reinstall_package")
@patch("geegoo_agent.update_cmd.refresh_bin_links")
def test_run_update_success(
    mock_links: MagicMock,
    mock_reinstall: MagicMock,
    mock_tar: MagicMock,
    tmp_path,
    monkeypatch,
    capsys,
) -> None:
    install_dir = tmp_path / "geegoo-agent"
    install_dir.mkdir()
    (install_dir / "pyproject.toml").write_text("[project]\nname='x'\n", encoding="utf-8")
    monkeypatch.setenv("GEEGOO_INSTALL_DIR", str(install_dir))

    code = run_update(method="tarball")
    out = capsys.readouterr().out

    assert code == 0
    assert "更新完成" in out
    mock_tar.assert_called_once()
    mock_reinstall.assert_called_once_with(install_dir, dev=True)
    mock_links.assert_called_once_with(install_dir)


@pytest.mark.unit
def test_sync_source_auto_uses_tarball_without_git(tmp_path, monkeypatch) -> None:
    install_dir = tmp_path / "install"
    install_dir.mkdir()
    called: list[str] = []

    def fake_tarball(path: Path, *, branch: str) -> str:
        called.append(branch)
        (path / "updated.txt").write_text("1", encoding="utf-8")
        return "tarball"

    monkeypatch.setattr("geegoo_agent.update_cmd._sync_via_tarball", fake_tarball)
    result = sync_source(install_dir, method="auto", branch="main")
    assert result == "tarball"
    assert called == ["main"]
