"""Unit tests for decision/meta read tools."""

from __future__ import annotations

from datetime import date, timedelta
from unittest.mock import MagicMock

import pytest

from geegoo_agent.infra.state_store import FileStateStore
from geegoo_agent.memory.working import WorkingMemoryStore
from geegoo_agent.tools.bootstrap import register_mvp_tools
from geegoo_agent.tools.registry import ToolRegistry
from geegoo_agent.tools.types import ToolCallRequest, ToolContext


@pytest.fixture
def registry() -> ToolRegistry:
    return register_mvp_tools(ToolRegistry())


@pytest.mark.unit
def test_recall_yesterday_summary_found(tmp_path, registry: ToolRegistry) -> None:
    yesterday = (date.today() - timedelta(days=1)).isoformat()
    report_dir = tmp_path / "reports" / yesterday
    report_dir.mkdir(parents=True)
    report_file = report_dir / "00700.HK-premarket.md"
    report_file.write_text("# 昨日报告\n内容", encoding="utf-8")

    ctx = ToolContext(
        session_id="sess-1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=MagicMock(),
    )
    result = registry.execute(
        ToolCallRequest(name="recall_yesterday_summary", arguments={"code": "00700.HK"}),
        ctx,
    )
    assert result.status == "ok"
    assert result.data["found"] is True
    assert "昨日报告" in result.data["summary"]


@pytest.mark.unit
def test_read_working_state_returns_summary(tmp_path, registry: ToolRegistry) -> None:
    store = WorkingMemoryStore(FileStateStore(tmp_path))
    store.create("sess-1", skill="pre_market")
    ctx = ToolContext(
        session_id="sess-1",
        mcp_token="mcp-test",
        dry_run=False,
        workspace_root=tmp_path,
        market_client=MagicMock(),
        working_store=store,
    )
    result = registry.execute(ToolCallRequest(name="read_working_state"), ctx)
    assert result.status == "ok"
    assert "phase=init" in result.summary
