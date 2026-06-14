"""Unit tests for SkillLoader and pre_market manifest."""

from __future__ import annotations

from pathlib import Path

import pytest

from geegoo_agent.exceptions import ConfigError
from geegoo_agent.runtime.skill_loader import SkillLoader

PROJECT_ROOT = Path(__file__).resolve().parents[2]

MVP_CORE_TOOLS = {
    "check_trading_day",
    "get_report_bot_codes",
    "get_mcp_analysis",
    "fetch_market_news",
    "fetch_stock_news",
    "get_capital_flow",
    "get_capital_distribution",
    "get_bot_yesterday_attitude",
    "create_pre_market_report",
    "save_local_report",
    "write_execution_log",
}


@pytest.fixture
def loader() -> SkillLoader:
    return SkillLoader(PROJECT_ROOT)


@pytest.mark.unit
def test_load_pre_market_manifest(loader: SkillLoader) -> None:
    manifest = loader.load("pre_market")
    assert manifest.name == "pre_market"
    assert manifest.version == "1.0.0"
    assert manifest.registered_tool_count == 16
    assert len(manifest.llm_tasks) == 3
    assert len(manifest.indices) == 5


@pytest.mark.unit
def test_manifest_tools_include_mvp_core(loader: SkillLoader) -> None:
    manifest = loader.load("pre_market")
    tool_set = set(manifest.tools)
    missing = MVP_CORE_TOOLS - tool_set
    assert not missing, f"manifest missing MVP tools: {missing}"


@pytest.mark.unit
def test_skill_asset_paths_exist(loader: SkillLoader) -> None:
    missing = loader.validate_asset_paths("pre_market")
    assert missing == [], f"missing skill assets: {missing}"


@pytest.mark.unit
def test_load_unknown_skill_raises(loader: SkillLoader) -> None:
    with pytest.raises(ConfigError, match="manifest not found"):
        loader.load("nonexistent_skill")


@pytest.mark.unit
def test_workflow_phases_defined(loader: SkillLoader) -> None:
    manifest = loader.load("pre_market")
    workflow = manifest.workflow
    assert "prelude" in workflow
    assert "phase_a" in workflow
    assert "phase_b" in workflow
    assert len(workflow["prelude"]) == 2
    assert workflow["phase_b"]["per_stock"]
