"""Default paths for user-level install (~/.geegoo, Hermes-style)."""

from __future__ import annotations

import os
from pathlib import Path


def geegoo_home() -> Path:
    return Path(os.environ.get("GEEGOO_HOME", Path.home() / ".geegoo"))


def default_install_dir() -> Path:
    return Path(os.environ.get("GEEGOO_INSTALL_DIR", geegoo_home() / "geegoo-agent"))


def default_config_path() -> Path:
    env = os.environ.get("GEEGOO_CONFIG", "").strip()
    if env:
        return Path(env)
    home_cfg = geegoo_home() / "config.json"
    if home_cfg.is_file():
        return home_cfg
    local = Path("config.json")
    if local.is_file():
        return local
    return home_cfg


def default_data_dir() -> Path:
    return geegoo_home() / "data"
