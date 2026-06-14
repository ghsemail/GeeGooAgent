"""Slash command completion tests."""

from __future__ import annotations

import pytest
from prompt_toolkit.document import Document

from geegoo_agent.runtime.chat_commands import SLASH_COMMANDS, build_help_text, slash_command_completer


@pytest.mark.unit
def test_build_help_text_lists_commands() -> None:
    text = build_help_text()
    assert "/help" in text
    assert "/model" in text
    assert "自动补全" in text


@pytest.mark.unit
def test_slash_completer_shows_all_on_bare_slash() -> None:
    completer = slash_command_completer()
    items = list(completer.get_completions(Document("/"), None))
    assert len(items) == len(SLASH_COMMANDS)


@pytest.mark.unit
def test_slash_completer_filters_prefix() -> None:
    completer = slash_command_completer()
    items = [c.text for c in completer.get_completions(Document("/mod"), None)]
    assert "/model" in items
    assert "/help" not in items
