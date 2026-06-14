"""Unit tests for sandbox guards."""

from __future__ import annotations

import pytest

from geegoo_agent.config import SandboxConfig
from geegoo_agent.exceptions import SandboxError
from geegoo_agent.infra.sandbox import NetworkPolicy, SandboxManager, WorkspaceGuard


@pytest.mark.unit
def test_workspace_resolve_relative_path(tmp_path) -> None:
    guard = WorkspaceGuard(tmp_path / "workspace")
    path = guard.resolve("artifacts/report.md")
    assert path.name == "report.md"
    assert path.parent.name == "artifacts"
    assert path.is_relative_to(guard.workspace_root)


@pytest.mark.unit
def test_workspace_rejects_parent_traversal(tmp_path) -> None:
    root = tmp_path / "workspace"
    guard = WorkspaceGuard(root)
    with pytest.raises(SandboxError, match="outside workspace"):
        guard.resolve("../outside.txt")


@pytest.mark.unit
def test_workspace_rejects_absolute_path(tmp_path) -> None:
    guard = WorkspaceGuard(tmp_path / "workspace")
    outside = (tmp_path / "outside_secret.txt").resolve()
    with pytest.raises(SandboxError, match="absolute paths"):
        guard.resolve(outside)


@pytest.mark.unit
def test_network_allows_listed_host() -> None:
    policy = NetworkPolicy(["118.195.135.97", "localhost"])
    policy.assert_host_allowed("http://118.195.135.97:5700/checkTradingDay")


@pytest.mark.unit
def test_network_rejects_unknown_host() -> None:
    policy = NetworkPolicy(["118.195.135.97"])
    with pytest.raises(SandboxError, match="not in allowlist"):
        policy.assert_host_allowed("http://evil.example.com/api")


@pytest.mark.unit
def test_sandbox_manager_exposes_workspace_and_network(tmp_path) -> None:
    mgr = SandboxManager(
        tmp_path / "ws",
        SandboxConfig(allowed_hosts=["127.0.0.1"]),
    )
    mgr.network.assert_host_allowed("http://127.0.0.1:5700/x")
    resolved = mgr.workspace.resolve("sessions/s1.json")
    assert "sessions" in str(resolved)
