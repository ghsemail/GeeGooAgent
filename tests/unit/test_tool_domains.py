"""Tests for tool domain taxonomy and chat allowlist."""

from __future__ import annotations

import pytest

from geegoo_agent.runtime.chat_tools import ON_DEMAND_CHAT_TOOLS
from geegoo_agent.tools.domains import (
    BOT_MANAGER_TOOLS,
    CHAT_ON_DEMAND_TOOLS,
    REMINDER_MANAGER_TOOLS,
    REPORT_WORKFLOW_TOOLS,
    tool_domain,
    ToolDomain,
)


@pytest.mark.unit
def test_report_workflow_tools_not_in_chat() -> None:
    chat = set(CHAT_ON_DEMAND_TOOLS)
    for name in REPORT_WORKFLOW_TOOLS:
        assert name not in chat, name


@pytest.mark.unit
def test_chat_includes_grid_reminder_list() -> None:
    assert "list_grid_reminders" in CHAT_ON_DEMAND_TOOLS
    assert "list_grid_bots" in CHAT_ON_DEMAND_TOOLS


@pytest.mark.unit
def test_get_report_bot_codes_is_report_workflow_only() -> None:
    assert tool_domain("get_report_bot_codes") is ToolDomain.REPORT_WORKFLOW
    assert "get_report_bot_codes" not in ON_DEMAND_CHAT_TOOLS


@pytest.mark.unit
def test_list_tools_have_manager_domains() -> None:
    assert tool_domain("list_grid_reminders") is ToolDomain.REMINDER_MANAGER
    assert tool_domain("list_grid_bots") is ToolDomain.BOT_MANAGER
    assert "list_grid_reminders" in REMINDER_MANAGER_TOOLS
    assert "list_grid_bots" in BOT_MANAGER_TOOLS
