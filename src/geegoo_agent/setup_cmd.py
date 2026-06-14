"""Interactive install + first-time configuration (``geegoo setup``)."""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
from getpass import getpass
from pathlib import Path
from typing import Any

from geegoo_agent.cli_meta import CLI_NAME
from geegoo_agent.config import AppConfig, load_config
from geegoo_agent.paths import default_data_dir
from geegoo_agent.exceptions import ConfigError
from geegoo_agent.llm.presets import LLMProviderName, PROVIDER_PRESETS
from geegoo_agent.paths import default_data_dir
from geegoo_agent.update_cmd import github_token, mask_github_token, save_github_token


def _default_template(project_root: Path) -> dict[str, Any]:
    example = project_root / "config.example.json"
    if example.is_file():
        return json.loads(example.read_text(encoding="utf-8"))
    return AppConfig(
        base_url="http://118.195.135.97:5700",
        api_key="sk-REPLACE",
        geegoo_url="http://118.195.135.97:5700",
        geegoo_api_key="sk-REPLACE",
        mcp_token="",
    ).model_dump(mode="json")


def _load_or_template(config_path: Path, project_root: Path) -> dict[str, Any]:
    if config_path.is_file():
        return json.loads(config_path.read_text(encoding="utf-8"))
    return _default_template(project_root)


def _sync_tradingbot_keys(raw: dict[str, Any], tradingbot: Path, *, host: str) -> None:
    from geegoo_agent.infra.tradingbot_sync import build_config

    synced = build_config(tradingbot.resolve(), base_host=host)
    for key in ("base_url", "api_key", "geegoo_url", "geegoo_api_key", "signal_base_url", "sandbox"):
        if key in synced:
            raw[key] = synced[key]


def _prompt_provider() -> LLMProviderName:
    print("\n选择 LLM 提供商：")
    names = list(PROVIDER_PRESETS.keys())
    for index, name in enumerate(names, start=1):
        preset = PROVIDER_PRESETS[name]
        print(f"  {index}. {preset.label}（默认模型 {preset.default_model}）")
    while True:
        choice = input("请输入序号 [1]: ").strip() or "1"
        if choice.isdigit() and 1 <= int(choice) <= len(names):
            return names[int(choice) - 1]
        if choice in PROVIDER_PRESETS:
            return choice  # type: ignore[return-value]
        print("无效选择，请重试。")


def _find_project_root(start: Path | None = None) -> Path:
    cursor = (start or Path.cwd()).resolve()
    for directory in (cursor, *cursor.parents):
        if (directory / "pyproject.toml").is_file() and (directory / "src" / "geegoo").is_dir():
            return directory
    return Path(__file__).resolve().parents[2]


def ensure_installed(project_root: Path, *, dev: bool = True) -> None:
    """Editable install so the ``geegoo`` command is available."""
    if not (project_root / "pyproject.toml").is_file():
        return
    target = "-e .[dev]" if dev else "-e ."
    print(f"正在安装依赖: pip install {target}")
    subprocess.check_call(
        [sys.executable, "-m", "pip", "install", "-e", ".[dev]" if dev else "."],
        cwd=project_root,
    )
    print(f"安装完成。可使用 `{CLI_NAME} run` / `{CLI_NAME} setup` 等命令。")


def _prompt_secret(label: str, *, current: str = "") -> str:
    hint = "（回车保留原值）" if current else ""
    value = getpass(f"{label}{hint}: ").strip()
    return value or current


def run_setup(
    config_path: str | Path,
    *,
    project_root: Path | None = None,
    provider: LLMProviderName | None = None,
    token_key: str | None = None,
    mcp_token: str | None = None,
    github_token_value: str | None = None,
    tradingbot: Path | None = None,
    host: str = "118.195.135.97",
    interactive: bool = True,
    skip_install: bool = False,
) -> Path:
    """Install package (editable) and create or update config.json."""
    root = project_root or _find_project_root()
    path = Path(config_path)
    if not path.is_absolute():
        path = root / path

    if not skip_install:
        ensure_installed(root, dev=True)

    if interactive:
        print(
            f"\n=== {CLI_NAME} setup ===\n"
            "配置 GeeGoo API、LLM（OpenAI / DeepSeek / Minimax）"
            "与 GitHub PAT（私有仓库 geegoo update）\n"
        )

    raw = _load_or_template(path, root)

    if tradingbot is not None:
        _sync_tradingbot_keys(raw, tradingbot, host=host)
        print(f"已从 TradingBot 同步 API Bearer → {path}")

    llm = raw.setdefault("llm", {})
    current_provider = llm.get("provider", "openai")
    current_token = llm.get("token_key", "")
    current_mcp = raw.get("mcp_token", "")

    if interactive and provider is None:
        provider = _prompt_provider()
    if provider is None:
        provider = current_provider if current_provider in PROVIDER_PRESETS else "openai"

    if interactive and not token_key:
        token_key = _prompt_secret("LLM token_key", current=current_token)
    if token_key is None:
        token_key = current_token

    if interactive and not mcp_token:
        hint = "（回车保留原值）" if current_mcp else ""
        entered = input(f"mcp_token{hint}: ").strip()
        mcp_token = entered or current_mcp
    if mcp_token is None:
        mcp_token = current_mcp

    current_github = github_token()
    if interactive and github_token_value is None:
        hint = f"（当前 {mask_github_token(current_github)}，回车保留）" if current_github else ""
        entered = getpass(f"GitHub PAT（私有仓库 geegoo update）{hint}: ").strip()
        github_token_value = entered or current_github
    elif github_token_value is None:
        github_token_value = current_github

    llm["provider"] = provider
    if token_key:
        llm["token_key"] = token_key
    if not llm.get("model"):
        llm["model"] = ""
    llm.pop("api_key_env", None)

    if mcp_token:
        raw["mcp_token"] = mcp_token

    github_token_path_written: Path | None = None
    if github_token_value:
        github_token_path_written = save_github_token(github_token_value)

    out_dir = str(raw.get("output_dir") or "").strip()
    if not out_dir or out_dir in {"./data", "data"}:
        raw["output_dir"] = str(default_data_dir())

    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(raw, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

    try:
        load_config(path)
    except ConfigError as exc:
        raise ConfigError(f"setup wrote invalid config: {exc}") from exc

    preset = PROVIDER_PRESETS[provider]
    model = llm.get("model") or preset.default_model
    print(f"\n已写入 {path}")
    print(f"  LLM: {preset.label} / {model}")
    print(f"  mcp_token: {raw.get('mcp_token') or '未配置'}")
    if github_token_path_written is not None:
        print(f"  GitHub PAT: 已写入 {github_token_path_written}（chmod 600）")
    elif github_token():
        print(f"  GitHub PAT: 已配置（{mask_github_token(github_token())}）")
    else:
        print("  GitHub PAT: 未配置 — 私有仓库 geegoo update 会失败")
    print(f"\n下一步:")
    print(f"  {CLI_NAME} doctor")
    print(f"  {CLI_NAME} chat")
    print(f"  {CLI_NAME} run pre_market --dry-run --config {path}")
    return path


def copy_example_if_missing(config_path: Path, project_root: Path) -> None:
    if config_path.is_file():
        return
    example = project_root / "config.example.json"
    if example.is_file():
        shutil.copy(example, config_path)
