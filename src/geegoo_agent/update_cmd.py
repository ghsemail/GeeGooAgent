"""Self-update for user-level installs (``geegoo update``, Hermes-style)."""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path
from typing import Literal

import httpx

from geegoo_agent.cli_meta import CLI_NAME
from geegoo_agent.paths import default_install_dir, geegoo_home

UpdateMethod = Literal["auto", "git", "tarball"]
_KEEP_ON_MERGE = frozenset({"venv", ".git"})


def default_repo_url() -> str:
    return os.environ.get("GEEGOO_REPO", "https://github.com/ghsemail/GeeGooAgent.git").strip()


def github_token() -> str:
    """Resolve token from env, ``~/.geegoo/github_token``, or ``~/.geegoo/env``."""
    token = os.environ.get("GEEGOO_GITHUB_TOKEN", "").strip()
    if token:
        return token

    token_file = github_token_path()
    if token_file.is_file():
        return token_file.read_text(encoding="utf-8").strip()

    env_file = geegoo_home() / "env"
    if env_file.is_file():
        for raw in env_file.read_text(encoding="utf-8").splitlines():
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("export "):
                line = line[len("export ") :]
            if line.startswith("GEEGOO_GITHUB_TOKEN="):
                value = line.split("=", 1)[1].strip().strip('"').strip("'")
                if value:
                    return value
    return ""


def github_token_path() -> Path:
    return geegoo_home() / "github_token"


def save_github_token(token: str) -> Path:
    """Persist PAT for private-repo ``geegoo update`` (``~/.geegoo/github_token``, mode 600)."""
    cleaned = token.strip()
    if not cleaned:
        raise ValueError("GitHub token is empty")
    if any(ch in cleaned for ch in "\n\r"):
        raise ValueError("GitHub token must be a single line")

    path = github_token_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(cleaned + "\n", encoding="utf-8")
    try:
        path.chmod(0o600)
    except OSError:
        pass
    return path


def mask_github_token(token: str) -> str:
    if len(token) <= 8:
        return "***"
    return f"{token[:4]}...{token[-4:]}"


def github_auth_headers() -> dict[str, str]:
    token = github_token()
    if not token:
        return {}
    return {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    }


def tarball_urls(branch: str = "main") -> list[str]:
    owner_repo = "ghsemail/GeeGooAgent"
    urls: list[str] = []
    custom = os.environ.get("GEEGOO_TARBALL_URL", "").strip()
    if custom:
        urls.append(custom)
    # api.github.com tarball supports private repos when GEEGOO_GITHUB_TOKEN is set.
    urls.append(f"https://api.github.com/repos/{owner_repo}/tarball/{branch}")
    if not github_token():
        urls.extend(
            [
                f"https://codeload.github.com/{owner_repo}/tar.gz/refs/heads/{branch}",
                f"https://github.com/{owner_repo}/archive/refs/heads/{branch}.tar.gz",
            ]
        )
    seen: set[str] = set()
    ordered: list[str] = []
    for url in urls:
        if url and url not in seen:
            seen.add(url)
            ordered.append(url)
    return ordered


def resolve_install_dir() -> Path:
    env = os.environ.get("GEEGOO_INSTALL_DIR", "").strip()
    if env:
        return Path(env)

    home_install = default_install_dir()
    if (home_install / "pyproject.toml").is_file():
        return home_install

    import geegoo_agent

    cursor = Path(geegoo_agent.__file__).resolve().parent
    for directory in (cursor, *cursor.parents):
        if (directory / "pyproject.toml").is_file() and (directory / "src" / "geegoo").is_dir():
            return directory
    return home_install


def _run(cmd: list[str], *, cwd: Path | None = None) -> None:
    subprocess.check_call(cmd, cwd=cwd)


def _venv_python(install_dir: Path) -> Path:
    if os.name == "nt":
        return install_dir / "venv" / "Scripts" / "python.exe"
    return install_dir / "venv" / "bin" / "python"


def _auth_repo_url(repo: str) -> str:
    token = github_token()
    if not token or not repo.startswith("https://"):
        return repo
    if "@" in repo.split("://", 1)[-1]:
        return repo
    return repo.replace("https://", f"https://x-access-token:{token}@", 1)


def _sync_via_git(install_dir: Path, *, branch: str, repo: str) -> str:
    if not (install_dir / ".git").is_dir():
        raise RuntimeError("not a git repository")

    auth_repo = _auth_repo_url(repo)
    origin = subprocess.run(
        ["git", "remote", "get-url", "origin"],
        cwd=install_dir,
        capture_output=True,
        text=True,
        check=False,
    )
    if origin.returncode != 0:
        _run(["git", "init"], cwd=install_dir)
        _run(["git", "remote", "add", "origin", auth_repo], cwd=install_dir)
    elif github_token() and origin.stdout.strip() != auth_repo:
        _run(["git", "remote", "set-url", "origin", auth_repo], cwd=install_dir)

    _run(["git", "fetch", "origin", branch], cwd=install_dir)
    _run(["git", "checkout", branch], cwd=install_dir)
    _run(["git", "pull", "--ff-only", "origin", branch], cwd=install_dir)
    return "git"


def _download_tarball(url: str) -> Path:
    tmp = tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False)
    path = Path(tmp.name)
    tmp.close()
    headers = github_auth_headers()
    with httpx.Client(follow_redirects=True, timeout=120.0, headers=headers) as client:
        with client.stream("GET", url) as response:
            response.raise_for_status()
            with path.open("wb") as handle:
                for chunk in response.iter_bytes():
                    handle.write(chunk)
    return path


def merge_tarball_into(install_dir: Path, tarball_path: Path) -> None:
    """Extract GitHub tarball and merge into install dir (keep venv/.git)."""
    install_dir.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as temp_name:
        temp_dir = Path(temp_name)
        with tarfile.open(tarball_path, "r:gz") as archive:
            archive.extractall(temp_dir)
        roots = [p for p in temp_dir.iterdir() if p.is_dir()]
        if not roots:
            raise RuntimeError("empty tarball")
        source_root = roots[0]

        for child in install_dir.iterdir():
            if child.name in _KEEP_ON_MERGE:
                continue
            if child.is_dir():
                shutil.rmtree(child)
            else:
                child.unlink()

        for child in source_root.iterdir():
            target = install_dir / child.name
            if child.is_dir():
                shutil.copytree(child, target)
            else:
                shutil.copy2(child, target)


def _sync_via_tarball(install_dir: Path, *, branch: str) -> str:
    errors: list[str] = []
    for url in tarball_urls(branch):
        try:
            archive = _download_tarball(url)
        except Exception as exc:
            errors.append(f"{url}: {exc}")
            continue
        try:
            merge_tarball_into(install_dir, archive)
        finally:
            archive.unlink(missing_ok=True)
        return f"tarball ({url})"

    if github_token():
        hint = "已设置 GEEGOO_GITHUB_TOKEN 仍失败：检查 token 是否有 repo 读权限、分支名是否正确。"
    else:
        hint = (
            "GeeGooAgent 是私有仓库，匿名下载会 404。"
            "请设置 GEEGOO_GITHUB_TOKEN，或写入 "
            f"{geegoo_home() / 'github_token'}（单行 token，chmod 600），"
            "或从开发机运行 scripts/ssh_upload_install.py 上传更新。"
        )
    detail = "; ".join(errors[:3])
    raise RuntimeError(f"all tarball URLs failed — {detail}. {hint}")


def sync_source(
    install_dir: Path,
    *,
    method: UpdateMethod = "auto",
    branch: str = "main",
    repo: str | None = None,
) -> str:
    repo_url = repo or default_repo_url()
    if method == "tarball":
        return _sync_via_tarball(install_dir, branch=branch)

    if method == "git":
        return _sync_via_git(install_dir, branch=branch, repo=repo_url)

    if (install_dir / ".git").is_dir():
        try:
            return _sync_via_git(install_dir, branch=branch, repo=repo_url)
        except (subprocess.CalledProcessError, RuntimeError):
            pass
    return _sync_via_tarball(install_dir, branch=branch)


def reinstall_package(install_dir: Path, *, dev: bool = True) -> None:
    python = _venv_python(install_dir)
    if not python.is_file():
        _run([sys.executable, "-m", "venv", str(install_dir / "venv")])
        python = _venv_python(install_dir)

    editable = f"{install_dir}[dev]" if dev else str(install_dir)
    _run([str(python), "-m", "pip", "install", "-U", "pip", "wheel", "-q"])
    _run([str(python), "-m", "pip", "install", "-q", "-e", editable])


def refresh_bin_links(install_dir: Path) -> None:
    bin_dir = geegoo_home() / "bin"
    bin_dir.mkdir(parents=True, exist_ok=True)
    venv_bin = install_dir / ("venv/Scripts" if os.name == "nt" else "venv/bin")
    for name in ("geegoo", "geegoo-agent"):
        src = venv_bin / (f"{name}.exe" if os.name == "nt" else name)
        if not src.is_file():
            continue
        link = bin_dir / (f"{name}.exe" if os.name == "nt" else name)
        if link.exists() or link.is_symlink():
            link.unlink()
        link.symlink_to(src)


def run_update(
    *,
    method: UpdateMethod = "auto",
    branch: str = "main",
    repo: str | None = None,
    skip_pip: bool = False,
    dev: bool = True,
) -> int:
    install_dir = resolve_install_dir()
    if not (install_dir / "pyproject.toml").is_file():
        print(
            f"未找到安装目录（{install_dir}）。"
            f"请先运行 install.sh 或设置 GEEGOO_INSTALL_DIR。",
            file=sys.stderr,
        )
        return 1

    print(f"==> {CLI_NAME} update")
    print(f"    dir:    {install_dir}")

    try:
        used = sync_source(install_dir, method=method, branch=branch, repo=repo)
        print(f"    source: {used}")
    except Exception as exc:
        print(f"更新源码失败: {exc}", file=sys.stderr)
        return 1

    if not skip_pip:
        try:
            print("==> pip install -e ...")
            reinstall_package(install_dir, dev=dev)
            refresh_bin_links(install_dir)
        except subprocess.CalledProcessError as exc:
            print(f"安装依赖失败: {exc}", file=sys.stderr)
            return 1

    print("\n更新完成。建议运行:")
    print(f"  {CLI_NAME} doctor")
    return 0
