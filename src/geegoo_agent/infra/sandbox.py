"""Sandbox policy guards (L0 MVP): workspace paths and network hosts."""

from __future__ import annotations

from pathlib import Path
from urllib.parse import urlparse

from geegoo_agent.config import SandboxConfig
from geegoo_agent.exceptions import SandboxError


class WorkspaceGuard:
    """Restrict file operations to a resolved workspace root."""

    def __init__(self, workspace_root: Path) -> None:
        self.workspace_root = workspace_root.resolve()
        self.workspace_root.mkdir(parents=True, exist_ok=True)

    def resolve(self, relative: str | Path) -> Path:
        rel = Path(relative)
        if rel.is_absolute() or rel.drive:
            raise SandboxError(f"absolute paths not allowed: {relative}")
        target = (self.workspace_root / rel).resolve()
        self.assert_in_workspace(target)
        return target

    def assert_in_workspace(self, path: Path) -> None:
        resolved = path.resolve()
        root = self.workspace_root
        try:
            resolved.relative_to(root)
        except ValueError as exc:
            raise SandboxError(f"path outside workspace: {path}") from exc


class NetworkPolicy:
    """HTTP host allowlist."""

    def __init__(self, allowed_hosts: list[str]) -> None:
        self.allowed_hosts = {h.lower() for h in allowed_hosts}

    def assert_host_allowed(self, url: str) -> None:
        parsed = urlparse(url)
        host = (parsed.hostname or "").lower()
        if not host:
            raise SandboxError(f"invalid url: {url}")
        if host not in self.allowed_hosts:
            raise SandboxError(f"host not in allowlist: {host}")


class SandboxManager:
    """Facade for workspace and network sandbox checks."""

    def __init__(self, workspace_root: Path, sandbox_config: SandboxConfig) -> None:
        self.workspace = WorkspaceGuard(workspace_root)
        self.network = NetworkPolicy(sandbox_config.allowed_hosts)
