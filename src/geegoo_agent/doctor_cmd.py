"""Health checks for ``geegoo doctor`` (Hermes-style diagnostics)."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import httpx

from geegoo_agent.config import AppConfig, load_config
from geegoo_agent.exceptions import ConfigError
from geegoo_agent.paths import default_install_dir, geegoo_home
from geegoo_agent.update_cmd import github_token, resolve_install_dir
from geegoo_agent.infra.secrets import ConfigSecrets
from geegoo_agent.llm.presets import PROVIDER_PRESETS, build_llm_provider
from geegoo_agent.llm.gateway import GatewayConfig, ModelGateway
from geegoo_agent.llm.cost import CostManager
from geegoo_agent.llm.types import Message


@dataclass
class CheckResult:
    name: str
    ok: bool
    detail: str


def check_config_file(path: Path) -> tuple[AppConfig | None, list[CheckResult]]:
    results: list[CheckResult] = []
    if not path.is_file():
        results.append(CheckResult("config", False, f"not found: {path}"))
        return None, results
    try:
        config = load_config(path)
    except ConfigError as exc:
        results.append(CheckResult("config", False, str(exc)))
        return None, results
    results.append(CheckResult("config", True, str(path.resolve())))
    return config, results


def check_secrets(config: AppConfig) -> list[CheckResult]:
    secrets = ConfigSecrets(config)
    results: list[CheckResult] = []
    checks: list[tuple[str, str, Any]] = [
        ("geegoo_api_key", "geegoo mcp api_key (sk-)", lambda: secrets.get("geegoo_api_key")),
        ("mcp_token", "mcp_token", lambda: secrets.get("mcp_token")),
        ("llm_token_key", "llm.token_key", lambda: secrets.get_llm_token_key()),
    ]
    for key, label, getter in checks:
        try:
            value = getter()
            detail = value if key == "mcp_token" else secrets.masked(key)
            results.append(CheckResult(label, True, detail))
        except ConfigError:
            results.append(CheckResult(label, False, "missing or placeholder — run geegoo setup"))
    return results


def _post_json(url: str, api_key: str, body: dict[str, Any], timeout: float = 20.0) -> tuple[int, str]:
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    try:
        with httpx.Client(timeout=timeout) as client:
            resp = client.post(url, json=body, headers=headers)
            preview = resp.text[:120].replace("\n", " ")
            return resp.status_code, preview
    except Exception as exc:
        return 0, str(exc)[:120]


def check_apis(config: AppConfig, secrets: ConfigSecrets) -> list[CheckResult]:
    results: list[CheckResult] = []
    mcp = secrets.get("mcp_token")
    geegoo = config.geegoo_url.rstrip("/")
    api_key = secrets.get("geegoo_api_key")

    code, preview = _post_json(
        f"{geegoo}/checkTradingDay",
        api_key,
        {"mcp_token": mcp, "code": "00700.HK"},
    )
    results.append(
        CheckResult(
            "geegoo mcp checkTradingDay",
            code == 200,
            f"HTTP {code} {preview}" if code else preview,
        )
    )

    code, preview = _post_json(
        f"{geegoo}/searchCode",
        api_key,
        {"regex": "00700", "market": ["HK"]},
    )
    results.append(
        CheckResult(
            "geegoo mcp searchCode",
            code == 200,
            f"HTTP {code} {preview}" if code else preview,
        )
    )

    code, preview = _post_json(
        f"{geegoo}/getCurrentPrice",
        api_key,
        {"mcp_token": mcp, "code": "00700.HK"},
    )
    results.append(
        CheckResult(
            "geegoo mcp getCurrentPrice",
            code == 200,
            f"HTTP {code} {preview}" if code else preview,
        )
    )
    return results


def check_github_update() -> list[CheckResult]:
    """Warn when private-repo self-update cannot work without a token."""
    install_dir = resolve_install_dir()
    has_git = (install_dir / ".git").is_dir()
    token = github_token()
    token_file = geegoo_home() / "github_token"
    if has_git and token:
        return [
            CheckResult(
                "geegoo update (git)",
                True,
                f"git repo + token ({token_file.name if token_file.is_file() else 'env'})",
            )
        ]
    if has_git:
        return [
            CheckResult(
                "geegoo update (git)",
                False,
                "私有仓库需 GEEGOO_GITHUB_TOKEN 或 ~/.geegoo/github_token",
            )
        ]
    if token:
        return [
            CheckResult(
                "geegoo update (tarball)",
                True,
                f"token set ({token_file.name if token_file.is_file() else 'env'})",
            )
        ]
    home_install = default_install_dir()
    if install_dir.resolve() == home_install.resolve():
        return [
            CheckResult(
                "geegoo update (tarball)",
                False,
                "未配置 token — 私有仓库 geegoo update 会 404；"
                f"写入 {token_file} 或运行 scripts/ssh_upload_install.py",
            )
        ]
    return [
        CheckResult(
            "geegoo update (tarball)",
            True,
            "dev install — 可用 git pull 或 ssh_upload_install.py",
        )
    ]


def check_llm(config: AppConfig, secrets: ConfigSecrets) -> list[CheckResult]:
    results: list[CheckResult] = []
    try:
        token_key = secrets.get_llm_token_key()
    except ConfigError as exc:
        results.append(CheckResult("LLM ping", False, str(exc)))
        return results

    preset = PROVIDER_PRESETS[config.llm.provider]
    provider = build_llm_provider(config.llm.provider, token_key, model=config.llm.model or None)
    gateway = ModelGateway(provider, CostManager(), GatewayConfig(max_retries=1))
    try:
        response = gateway.chat(
            [Message(role="user", content="reply with exactly: ok")],
            [],
            session_id="doctor",
            step=0,
        )
        text = (response.content or "").strip()[:80]
        results.append(
            CheckResult(
                f"LLM {preset.label}",
                bool(text),
                text or "empty response",
            )
        )
    except Exception as exc:
        results.append(CheckResult(f"LLM {preset.label}", False, str(exc)[:120]))
    return results


def run_doctor(
    config_path: str | Path,
    *,
    skip_llm: bool = False,
    skip_api: bool = False,
) -> int:
    path = Path(config_path)
    config, results = check_config_file(path)
    _print_results(results)
    if config is None:
        return 1

    secret_results = check_secrets(config)
    _print_results(secret_results)
    _print_results(check_github_update())
    if any(not r.ok for r in secret_results):
        print("\n提示: 运行 geegoo setup 填写 mcp_token、llm.token_key、GitHub PAT 与 API Bearer。")
        return 1

    secrets = ConfigSecrets(config)
    if not skip_api:
        api_results = check_apis(config, secrets)
        _print_results(api_results)
        if any(not r.ok for r in api_results):
            return 1

    if not skip_llm:
        llm_results = check_llm(config, secrets)
        _print_results(llm_results)
        if any(not r.ok for r in llm_results):
            return 1

    print("\n全部检查通过。可运行: geegoo chat")
    return 0


def _print_results(results: list[CheckResult]) -> None:
    for row in results:
        mark = "OK" if row.ok else "FAIL"
        print(f"  [{mark}] {row.name}: {row.detail}")
