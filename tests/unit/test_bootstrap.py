"""Bootstrap registration aligned with manifest."""

from __future__ import annotations

from pathlib import Path

import pytest

from geegoo_agent.runtime.skill_loader import SkillLoader
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry

PROJECT_ROOT = Path(__file__).resolve().parents[2]


@pytest.mark.unit
def test_bootstrap_registers_all_manifest_tools() -> None:
    manifest = SkillLoader(PROJECT_ROOT).load("pre_market")
    registry = register_mvp_tools(ToolRegistry(), project_root=PROJECT_ROOT)
    registered = set(registry.list_names())
    missing = set(manifest.tools) - registered
    assert not missing, f"manifest tools not registered: {missing}"


@pytest.mark.unit
def test_manifest_tool_count_is_sixteen() -> None:
    manifest = SkillLoader(PROJECT_ROOT).load("pre_market")
    assert len(manifest.tools) == 16
